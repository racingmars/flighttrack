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

func loadFlightaware(db *sqlx.DB) error {
	err := createTempFlightawareTable(db)
	if err != nil {
		return err
	}
	defer dropTempFlightaware(db)

	if err = loadFlightawareData(db); err != nil {
		log.Error().Err(err).Msgf("Error loading Flightaware data")
		return err
	}

	if err = mergeFlightawareData(db); err != nil {
		log.Error().Err(err).Msgf("Error inserting Flightaware data")
		return err
	}

	return nil
}

func createTempFlightawareTable(db *sqlx.DB) error {
	sqlstmtFA := `
		CREATE TEMPORARY TABLE flightaware (
			icao CHAR(6) PRIMARY KEY,
			registration VARCHAR(10),
			type VARCHAR(10)
		)`

	_, err := db.Exec(sqlstmtFA)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't create Flightaware temp table")
		return err
	}

	return nil
}

func dropTempFlightaware(db *sqlx.DB) {
	_, err := db.Exec(`DROP TABLE flightaware`)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't drop flightaware")
	}
}

func loadFlightawareData(db *sqlx.DB) error {
	f, err := os.Open("data/flightaware-20180720.csv.bz2")
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't open Flightaware file")
		return err
	}
	defer f.Close()

	rdr := csv.NewReader(bzip2.NewReader(f))

	txn, err := db.Begin()
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't open transaction")
		return err
	}

	stmt, err := txn.Prepare(pq.CopyIn("flightaware", "icao", "registration", "type"))
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't prepare Flightaware insert statement")
		return err
	}

	rowCount := 0

	for {
		row, err := rdr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error().Err(err).Msgf("Error reading Flightaware file")
			stmt.Close()
			txn.Rollback()
			return err
		}

		rowCount++

		icao := row[0]
		registration := row[1]
		typecode := row[2]

		_, err = stmt.Exec(icao, registration, typecode)
		if err != nil {
			log.Error().Err(err).Msgf("Couldn't insert %s into flightaware", icao)
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

	log.Info().Msgf("Inserted %d Flightaware registrations", rowCount)

	return nil
}

func mergeFlightawareData(db *sqlx.DB) error {
	stmt := `
		INSERT INTO registration
		(icao, registration, typecode, source)
		SELECT
			LOWER(icao), registration, NULLIF(type, ''), 'FA'
		FROM flightaware
		ON CONFLICT (icao) DO UPDATE SET typecode=EXCLUDED.typecode WHERE registration.typecode IS NULL`
	log.Info().Msgf("About to merge Flightaware data into registration table")
	result, err := db.Exec(stmt)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't insert Flightaware data")
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil {
		log.Info().Msgf("Inserted %d Flightaware registration records", rows)
	}
	return nil
}
