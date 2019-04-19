package data

import (
	"database/sql"
)

type Registration struct {
	Icao, Source string
	Registration sql.NullString
	Typecode     sql.NullString
	Mfg          sql.NullString
	Model        sql.NullString
	Owner        sql.NullString
	City         sql.NullString
	State        sql.NullString
	Country      sql.NullString
	Year         sql.NullInt64
}

func (d *DAO) GetRegistration(icao string) (Registration, error) {
	reg := Registration{}
	err := d.db.Get(&reg,
		`SELECT icao, registration, typecode, mfg, model, owner, city, state, country, source,
				CASE
				  WHEN year IS NULL THEN null
				  WHEN year < 1850 THEN null
				  ELSE year
				END AS year
			FROM registration
			WHERE icao=$1`, icao)
	return reg, err
}

// SearchRegistration searches for the given registration (query) and, if
// found, returns the ICAO ID of the registration record. If not found,
// returns empty string. Error is non-nil only for database errors, NOT
// for no record found.
func (d *DAO) SearchRegistration(query string) (string, error) {
	var icao string
	err := d.db.Get(&icao, `SELECT icao FROM registration WHERE registration=UPPER($1)`, query)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return icao, nil
}
