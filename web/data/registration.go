package data

import "database/sql"

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
