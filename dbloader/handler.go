package main

import (
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/racingmars/flighttrack/decoder"
	"github.com/racingmars/flighttrack/tracker"
)

type handler struct {
	db         *sqlx.DB
	idmap      map[string]int
	currentTxn *sqlx.Tx
	logstmt    *sqlx.Stmt
	batchCount int
}

func newHandler(db *sqlx.DB) *handler {
	tx, err := db.Beginx()
	if err != nil {
		log.Panic().Err(err).Msgf("couldn't make new transaction in newHandler()")
	}
	return &handler{
		db:         db,
		idmap:      make(map[string]int),
		currentTxn: tx,
	}
}

func newHandlerWithState(db *sqlx.DB, handlerstate []byte) (*handler, error) {
	tx, err := db.Beginx()
	if err != nil {
		return nil, err
	}
	var idmap map[string]int
	err = json.Unmarshal(handlerstate, &idmap)
	if err != nil {
		return nil, err
	}
	return &handler{
		db:         db,
		idmap:      idmap,
		currentTxn: tx,
	}, nil
}

func (h *handler) Close() {
	if h.logstmt != nil {
		h.logstmt.Close()
	}
	if h.currentTxn != nil {
		err := h.currentTxn.Commit()
		if err != nil {
			log.Error().Err(err).Msg("couldn't commit transaction when closing handler")
		}
	}
}

func (h *handler) NewFlight(icaoID string, firstSeen time.Time) {
	row := h.db.QueryRow("INSERT INTO flight (icao, first_seen) VALUES ($1, $2) RETURNING id", icaoID, firstSeen.UTC())
	var id int
	err := row.Scan(&id)
	if err != nil {
		log.Error().Err(err).Msgf("couldn't get inserted ID for flight %s", icaoID)
		return
	}
	h.idmap[icaoID] = id
	h.batchCount++
}

func (h *handler) CloseFlight(icaoID string, lastSeen time.Time, messages int) {
	id, ok := h.idmap[icaoID]
	if !ok {
		log.Error().Msgf("couldn't find id for flight %s", icaoID)
		return
	}

	_, err := h.db.Exec("UPDATE flight SET last_seen=$1, msg_count=$2 WHERE id=$3", lastSeen.UTC(), messages, id)
	if err != nil {
		log.Error().Err(err).Msgf("closing flight %s (%d)", icaoID, id)
	}

	delete(h.idmap, icaoID)
	h.batchCount++
}

func (h *handler) SetIdentity(icaoID, callsign string, category decoder.AircraftType, change bool) {
	var err error
	id, ok := h.idmap[icaoID]
	if !ok {
		log.Error().Msgf("couldn't find id for flight %s", icaoID)
		return
	}

	if change {
		// Keep the flight set to the original callsign we saw, but indicate we've seen a change
		_, err = h.db.Exec("UPDATE flight SET multicall=true WHERE id=$1", id)
	} else {
		_, err = h.db.Exec("UPDATE flight SET callsign=$1, category=$2 WHERE id=$3", callsign, category, id)
	}

	if err != nil {
		log.Error().Err(err).Msgf("setting callsign for flight %s/%d (%d)", icaoID, category, id)
	}
	h.batchCount++
}

func (h *handler) AddTrackPoint(icaoID string, t tracker.TrackLog) {
	if h.logstmt == nil {
		stmt, err := h.currentTxn.Preparex(`INSERT INTO tracklog (flight_id, time, latitude, longitude, heading, speed, altitude, vs, callsign, category)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);`)
		if err != nil {
			log.Error().Err(err).Msgf("preparing tracklog statement")
			return
		}
		h.logstmt = stmt
	}

	id, ok := h.idmap[icaoID]
	if !ok {
		log.Error().Msgf("couldn't find id for flight %s", icaoID)
		return
	}

	var heading, vs, altitude, speed *int
	var latitude, longitude *float64
	var callsign *string
	var category *decoder.AircraftType

	if t.HeadingValid {
		heading = &t.Heading
	}
	if t.VSValid {
		vs = &t.VS
	}
	if t.AltitudeValid {
		altitude = &t.Altitude
	}
	if t.SpeedValid {
		speed = &t.Speed
	}
	if t.PositionValid {
		latitude = &t.Latitude
		longitude = &t.Longitude
	}
	if t.IdentityValid {
		callsign = &t.Callsign
		category = &t.Category
	}

	_, err := h.logstmt.Exec(id, t.Time.UTC(), latitude, longitude, heading, speed, altitude, vs, callsign, category)

	if err != nil {
		log.Error().Err(err).Msgf("adding track log for flight %s (%d)", icaoID, id)
	}

	h.batchCount++
	if h.batchCount > 25000 {
		h.Flush()
	}
}

func (h *handler) Flush() {
	//log.Debug().Msg("committing transaction")
	if h.logstmt != nil {
		h.logstmt.Close()
		h.logstmt = nil
	}
	err := h.currentTxn.Commit()
	if err != nil {
		log.Error().Err(err).Msgf("couldn't commit transaction")
	}
	tx, err := h.db.Beginx()
	if err != nil {
		log.Panic().Err(err).Msg("couldn't start new transaction")
	}
	h.currentTxn = tx
	h.batchCount = 0
}

func (h *handler) GetState() []byte {
	data, err := json.Marshal(h.idmap)
	if err != nil {
		log.Error().Err(err).Msg("Unable to marshal handler state")
		return nil
	}
	return data
}

func (h *handler) saveState(trackerstate []byte, lastRawMessageID int64) {
	handlerstate := h.GetState()

	if !(trackerstate != nil && handlerstate != nil && lastRawMessageID > 0) {
		log.Error().Msg("Can't save state; inputs are not valid")
		return
	}

	txn, err := h.db.Begin()
	if err != nil {
		log.Error().Err(err).Msg("Couldn't open transaction to save handler state")
		return
	}

	_, err = txn.Exec(
		`INSERT INTO parameters (name, value_txt) VALUES ('trackerstate', $1)
		 ON CONFLICT (name)
		 DO UPDATE SET value_txt = EXCLUDED.value_txt`,
		string(trackerstate))
	if err != nil {
		log.Error().Err(err).Msg("Couldn't insert tracker state")
		txn.Rollback()
		return
	}

	_, err = txn.Exec(
		`INSERT INTO parameters (name, value_txt) VALUES ('handlerstate', $1)
		 ON CONFLICT (name)
		 DO UPDATE SET value_txt = EXCLUDED.value_txt`,
		string(handlerstate))
	if err != nil {
		log.Error().Err(err).Msg("Couldn't insert handler state")
		txn.Rollback()
		return
	}

	_, err = txn.Exec(
		`INSERT INTO parameters (name, value_int) VALUES ('lastmsgid', $1)
		 ON CONFLICT (name)
		 DO UPDATE SET value_int = EXCLUDED.value_int`,
		lastRawMessageID)
	if err != nil {
		log.Error().Err(err).Msg("Couldn't insert last message ID")
		txn.Rollback()
		return
	}

	log.Debug().Msgf("Saving state with last message ID: %d", lastRawMessageID)
	err = txn.Commit()
	if err != nil {
		log.Error().Err(err).Msg("Couldn't commit transaction to save state")
	}
}
