package main

import (
	"bufio"
	"compress/bzip2"
	"encoding/csv"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

func loadFAA(db *sqlx.DB) error {
	err := createTempFAATables(db)
	if err != nil {
		return err
	}
	defer dropTempFAATables(db)

	if err = loadFAAAircraftModels(db); err != nil {
		log.Error().Err(err).Msgf("Error loading FAA aircraft models")
		return err
	}

	if err = loadFAARegistrations(db); err != nil {
		log.Error().Err(err).Msgf("Error loading FAA registrations")
		return err
	}

	if err = mergeFAAData(db); err != nil {
		log.Error().Err(err).Msgf("Error inserting FAA data")
		return err
	}

	return nil
}

func createTempFAATables(db *sqlx.DB) error {
	sqlstmtAircraft := `
		CREATE TEMPORARY TABLE faa_acft (
			model_code TEXT PRIMARY KEY,
			mfg TEXT,
			model TEXT
		)`
	sqlstmtReg := `
		CREATE TEMPORARY TABLE faa_reg (
			nnumber TEXT PRIMARY KEY,
			model_code TEXT NOT NULL REFERENCES faa_acft(model_code),
			year INTEGER NOT NULL,
			owner TEXT,
			city TEXT,
			state TEXT,
			country TEXT,
			icao CHAR(6) NOT NULL
		)`

	_, err := db.Exec(sqlstmtAircraft)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't create FAA aircraft model temp table")
		return err
	}
	_, err = db.Exec(sqlstmtReg)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't create FAA registration temp table")
		return err
	}

	return nil
}

func dropTempFAATables(db *sqlx.DB) {
	_, err := db.Exec(`DROP TABLE faa_reg`)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't drop faa_reg")
	}
	_, err = db.Exec(`DROP TABLE faa_acft`)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't drop faa_acft")
	}
}

func loadFAAAircraftModels(db *sqlx.DB) error {
	f, err := os.Open("data/faa/ACFTREF.txt.bz2")
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't open FAA aircraft file")
		return err
	}
	defer f.Close()

	rdr := bufio.NewScanner(bzip2.NewReader(f))

	txn, err := db.Begin()
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't open transaction")
		return err
	}

	stmt, err := txn.Prepare(pq.CopyIn("faa_acft", "model_code", "mfg", "model"))
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't prepare faa_acft insert statement")
		return err
	}

	// header
	rdr.Scan()

	rowCount := 0

	for rdr.Scan() {
		row := strings.Split(rdr.Text(), ",")

		rowCount++

		code := strings.TrimSpace(row[0])
		mfr := strings.TrimSpace(row[1])
		model := strings.TrimSpace(row[2])

		_, err = stmt.Exec(code, mfr, model)
		if err != nil {
			log.Error().Err(err).Msgf("Couldn't insert %s into faa_acft", code)
			continue
		}
	}
	if err := rdr.Err(); err != nil {
		log.Error().Err(err).Msgf("Error reading FAA aircrat model file")
		stmt.Close()
		txn.Rollback()
		return err
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

	return nil
}

func loadFAARegistrations(db *sqlx.DB) error {
	f, err := os.Open("data/faa/MASTER.txt.bz2")
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't open FAA registration file")
		return err
	}
	defer f.Close()

	rdr := csv.NewReader(bzip2.NewReader(f))
	rdr.ReuseRecord = true
	rdr.LazyQuotes = true

	stmt, err := db.Preparex("INSERT INTO faa_reg VALUES ('N'||$1, $2, $3, $4, $5, $6, $7, $8)")
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't prepare faa_acft insert statement")
		return err
	}
	defer stmt.Close()

	// header
	rdr.Read()

	for {
		row, err := rdr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error().Err(err).Msgf("Couldn't read/parse from faa aircraft file")
			return err
		}

		nnumber := strings.TrimSpace(row[0])
		modelcode := strings.TrimSpace(row[2])
		yearstr := strings.TrimSpace(row[4])
		owner := strings.TrimSpace(row[6])
		city := strings.TrimSpace(row[9])
		state := strings.TrimSpace(row[10])
		country := strings.TrimSpace(row[14])
		icao := strings.TrimSpace(row[33])

		year, err := strconv.Atoi(yearstr)
		if err != nil {
			log.Error().Err(err).Msgf("Couldn't convert `%s` to integer", yearstr)
			continue
		}

		_, err = stmt.Exec(nnumber, modelcode, year, owner, city, state, country, icao)
		if err != nil {
			log.Error().Err(err).Msgf("Couldn't insert %s into faa_reg", nnumber)
			continue
		}
	}

	return nil
}

func mergeFAAData(db *sqlx.DB) error {
	stmt := `
		INSERT INTO registration
		(icao, registration, mfg, model, year, owner, city, state, country, source)
		SELECT
			r.icao, r.nnumber, a.mfg, a.model, r.year, r.owner, r.city, r.state, r.country, 'FAA'
		FROM faa_reg r
		INNER JOIN faa_acft a ON r.model_code=a.model_code`
	log.Info().Msgf("About to merge FAA data into registration table")
	result, err := db.Exec(stmt)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't insert FAA data")
		return err
	}
	log.Info().Msgf("Inserted %d FAA registration records", result.RowsAffected)
	return nil
}
