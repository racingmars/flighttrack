package main

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/racingmars/flighttrack/tracker"
)

type handler struct {
	db      *sqlx.DB
	idmap   map[string]int
	logstmt *sqlx.Stmt
}

func newHandler(db *sqlx.DB) *handler {
	return &handler{
		db:    db,
		idmap: make(map[string]int),
	}
}

func (h *handler) Close() {
	if h.logstmt != nil {
		h.logstmt.Close()
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
}

func (h *handler) SetIdentity(icaoID, callsign string, change bool) {
	var err error
	id, ok := h.idmap[icaoID]
	if !ok {
		log.Error().Msgf("couldn't find id for flight %s", icaoID)
		return
	}

	if change {
		_, err = h.db.Exec("UPDATE flight SET callsign=$1, multicall=true WHERE id=$2", callsign, id)
	} else {
		_, err = h.db.Exec("UPDATE flight SET callsign=$1 WHERE id=$2", callsign, id)
	}

	if err != nil {
		log.Error().Err(err).Msgf("setting callsign for flight %s (%d)", icaoID, id)
	}
}

func (h *handler) AddTrackPoint(icaoID string, t tracker.TrackLog) {
	if h.logstmt == nil {
		stmt, err := h.db.Preparex(`INSERT INTO tracklog (flight_id, time, latitude, longitude, heading, speed, altitude, vs)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`)
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

	_, err := h.logstmt.Exec(id, t.Time.UTC(), latitude, longitude, heading, speed, altitude, vs)

	if err != nil {
		log.Error().Err(err).Msgf("adding track log for flight %s (%d)", icaoID, id)
	}

}
