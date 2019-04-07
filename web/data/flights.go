package data

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type Flight struct {
	ID             int            `db:"id"`
	Icao           string         `db:"icao"`
	Callsign       sql.NullString `db:"callsign"`
	FirstSeen      time.Time      `db:"first_seen"`
	LastSeen       pq.NullTime    `db:"last_seen"`
	MsgCount       sql.NullInt64  `db:"msg_count"`
	Registration   sql.NullString
	Owner          sql.NullString
	Airline        sql.NullString `db:"airline"`
	TypeCode       sql.NullString `db:"typecode"`
	MfgYear        sql.NullInt64  `db:"year"`
	Mfg            sql.NullString
	Model          sql.NullString
	Icon           string
	IconX          int
	IconY          int
	Category       sql.NullInt64 `db:"category"`
	CategoryString string
}

type TrackLog struct {
	ID                           int `db:"id"`
	Time                         time.Time
	Latitude, Longitude          sql.NullFloat64
	Heading, Speed, Altitude, Vs sql.NullInt64
	Callsign                     sql.NullString
}

const baseFlightQuery = `
	SELECT f.id, f.icao, f.callsign, f.first_seen, f.last_seen, f.msg_count, f.category,
		   r.registration, r.owner, a.name AS airline, r.typecode, r.mfg, r.model,
		   CASE
			 WHEN r.year IS NULL THEN null
			 WHEN r.year < 1850 THEN null
			 ELSE r.year
		   END AS year
	FROM flight f
	LEFT OUTER JOIN registration r ON f.icao=r.icao
	LEFT OUTER JOIN airline a ON a.icao=substring(f.callsign from 1 for 3) AND f.icao NOT LIKE 'ae%'
	`

func (d *DAO) GetFlight(id int) (Flight, error) {
	flight := Flight{}
	err := d.db.Get(&flight,
		baseFlightQuery+
			`WHERE f.id = $1`,
		id)
	return flight, err
}

func (d *DAO) GetFlightsForDateRange(start, end time.Time) ([]Flight, error) {
	flights := make([]Flight, 0)
	err := d.db.Select(&flights, baseFlightQuery+
		`WHERE f.first_seen >= $1 AND f.first_seen < $2
		 ORDER BY f.first_seen`, start, end)
	return flights, err
}

func (d *DAO) GetFlightsActive() ([]Flight, error) {
	flights := make([]Flight, 0)
	err := d.db.Select(&flights, baseFlightQuery+
		`WHERE f.last_seen IS NULL
		 ORDER BY f.first_seen`)
	return flights, err
}

func (d *DAO) GetFlightsForAirframe(icao string) ([]Flight, error) {
	flights := make([]Flight, 0)
	err := d.db.Select(&flights, baseFlightQuery+
		`WHERE f.icao = $1
		 ORDER BY f.first_seen`, icao)
	return flights, err
}

func (d *DAO) GetTrackLog(flightID int) ([]TrackLog, error) {
	tracklog := make([]TrackLog, 0)
	err := d.db.Select(&tracklog,
		`SELECT id, time, latitude, longitude, heading, speed, altitude, vs, callsign
	 	 FROM tracklog
		 WHERE flight_id=$1
		 ORDER BY time`, flightID)
	return tracklog, err
}
