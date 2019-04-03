package main

import (
	"compress/bzip2"
	"encoding/csv"
	"io"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

func loadAircraftDB(db *sqlx.DB) error {
	err := createTempAircraftDBTable(db)
	if err != nil {
		return err
	}
	defer dropTempAircraftDB(db)

	if err = loadAircraftDBData(db); err != nil {
		log.Error().Err(err).Msgf("Error loading AircraftDB data")
		return err
	}

	if err = mergeAircraftDBData(db); err != nil {
		log.Error().Err(err).Msgf("Error inserting AircraftDB data")
		return err
	}

	return nil
}

func createTempAircraftDBTable(db *sqlx.DB) error {
	sqlstmtFA := `
		CREATE TEMPORARY TABLE aircraftdb (
			icao CHAR(6) PRIMARY KEY,
			regid VARCHAR(10),
			mdl VARCHAR(4),
			type TEXT,
			operator TEXT
		)`

	_, err := db.Exec(sqlstmtFA)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't create aircraftdb temp table")
		return err
	}

	return nil
}

func dropTempAircraftDB(db *sqlx.DB) {
	_, err := db.Exec(`DROP TABLE aircraftdb`)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't drop aircraftdb")
	}
}

func loadAircraftDBData(db *sqlx.DB) error {
	f, err := os.Open("data/aircraft_db/aircraft_db.csv.bz2")
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't open AircraftDB file")
		return err
	}
	defer f.Close()

	rdr := csv.NewReader(bzip2.NewReader(f))

	txn, err := db.Begin()
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't open transaction")
		return err
	}

	stmt, err := txn.Prepare(pq.CopyIn("aircraftdb", "icao", "regid", "mdl", "type", "operator"))
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't prepare AircraftDB insert statement")
		return err
	}

	rowCount := 0

	for {
		row, err := rdr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error().Err(err).Msgf("Error reading AircraftDB file")
			stmt.Close()
			txn.Rollback()
			return err
		}

		rowCount++

		icao := row[0]
		regid := row[1]
		mdl := row[2]
		typecode := row[3]
		operator := row[4]

		if len(regid) > 10 {
			log.Warn().Msgf("`%s` has too long regid `%s`", icao, regid)
			continue
		}

		_, err = stmt.Exec(icao, regid, mdl, typecode, operator)
		if err != nil {
			log.Error().Err(err).Msgf("Couldn't insert %s into AircraftDB", icao)
			return err
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		log.Error().Err(err).Msgf("Error flushing statement")
		txn.Rollback()
		return err
	}

	err = stmt.Close()
	if err != nil {
		log.Error().Err(err).Msgf("Error closing statement")
		txn.Rollback()
		return err
	}

	err = txn.Commit()
	if err != nil {
		log.Error().Err(err).Msgf("Error committing transaction")
	}

	log.Info().Msgf("Inserted %d AircraftDB registrations", rowCount)

	return nil
}

func mergeAircraftDBData(db *sqlx.DB) error {
	stmt := `
		INSERT INTO registration
		(icao, registration, typecode, owner, source)
		SELECT
			LOWER(icao), UPPER(regid), NULLIF(UPPER(mdl), ''), operator, 'ADB'
		FROM aircraftdb
		ON CONFLICT (icao) DO UPDATE SET typecode=EXCLUDED.typecode WHERE registration.typecode IS NULL`
	log.Info().Msgf("About to merge AircraftDB data into registration table")
	result, err := db.Exec(stmt)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't insert AircraftDB data")
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil {
		log.Info().Msgf("Inserted %d AircraftDB registration records", rows)
	}
	return nil
}
