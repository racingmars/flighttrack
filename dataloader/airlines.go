package main

import (
	"compress/bzip2"
	"encoding/csv"
	"io"
	"os"
	"regexp"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

func loadAirlines(db *sqlx.DB) error {
	_, err := db.Exec("TRUNCATE TABLE airline")
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't truncate airline table")
		return err
	}

	if err = loadAirlineData(db); err != nil {
		log.Error().Err(err).Msgf("Error loading Airline data")
		return err
	}
	return nil
}

func loadAirlineData(db *sqlx.DB) error {
	airlines := make(map[string]string)

	f, err := os.Open("data/airlines.dat.bz2")
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't open airlines file")
		return err
	}
	defer f.Close()

	rdr := csv.NewReader(bzip2.NewReader(f))

	txn, err := db.Begin()
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't open transaction")
		return err
	}

	stmt, err := txn.Prepare(pq.CopyIn("airline", "name", "icao", "callsign", "country"))
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't prepare airlines insert statement")
		return err
	}

	rowCount := 0
	insertCount := 0

	validpattern := regexp.MustCompile(`[A-Z]{3}`)

	for {
		row, err := rdr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error().Err(err).Msgf("Error reading airlines file")
			stmt.Close()
			txn.Rollback()
			return err
		}

		rowCount++

		icao := row[4]
		name := row[1]
		callsign := row[5]
		country := row[6]
		active := row[7]

		if active != "Y" {
			continue
		}

		if !validpattern.MatchString(icao) {
			log.Debug().Msgf("Skipping `%s` (`%s`)", icao, name)
			continue
		}

		if oldname, ok := airlines[icao]; ok {
			// duplicate...we already used this code
			log.Warn().Msgf("Code `%s` for `%s` already used for `%s`", icao, name, oldname)
			continue
		}
		airlines[icao] = name

		insertCount++

		dbName := &name
		dbCallsign := &callsign
		dbCountry := &country

		if name == "" || name == "\\N" {
			dbName = nil
		}

		if callsign == "" || callsign == "\\N" {
			dbCallsign = nil
		}

		if country == "" || country == "\\N" {
			dbCountry = nil
		}

		_, err = stmt.Exec(dbName, icao, dbCallsign, dbCountry)
		if err != nil {
			log.Error().Err(err).Msgf("Couldn't insert %s into airline", icao)
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

	log.Info().Msgf("Inserted %d (of %d) airline registrations", insertCount, rowCount)

	return nil
}
