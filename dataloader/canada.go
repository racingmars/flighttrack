package main

import (
	"compress/bzip2"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

func loadCanada(db *sqlx.DB) error {
	err := createTempCanadaTables(db)
	if err != nil {
		return err
	}
	defer dropTempCanadaTables(db)

	if err = loadCanadaOwners(db); err != nil {
		log.Error().Err(err).Msgf("Error loading Canada owners")
		return err
	}

	if err = loadCanadaRegistrations(db); err != nil {
		log.Error().Err(err).Msgf("Error loading Canada registrations")
		return err
	}

	if err = mergeCanadaData(db); err != nil {
		log.Error().Err(err).Msgf("Error inserting Canada data")
		return err
	}

	return nil
}

func createTempCanadaTables(db *sqlx.DB) error {
	sqlstmtOwners := `
		CREATE TEMPORARY TABLE canada_owner (
			mark VARCHAR(4) PRIMARY KEY,
			owner VARCHAR(120),
			city VARCHAR(40),
			province VARCHAR(100),
			country VARCHAR(100)
		)`
	sqlstmtReg := `
		CREATE TEMPORARY TABLE canada_reg (
			mark TEXT PRIMARY KEY,
			mfg TEXT,
			model TEXT,
			year INT,
			icao CHAR(6) NOT NULL,
			marktrim TEXT NOT NULL
		)`

	_, err := db.Exec(sqlstmtOwners)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't create Canada owners temp table")
		return err
	}
	_, err = db.Exec(sqlstmtReg)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't create Canada registration temp table")
		return err
	}

	return nil
}

func dropTempCanadaTables(db *sqlx.DB) {
	_, err := db.Exec(`DROP TABLE canada_owner`)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't drop faa_reg")
	}
	_, err = db.Exec(`DROP TABLE canada_reg`)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't drop faa_acft")
	}
}

func loadCanadaOwners(db *sqlx.DB) error {
	f, err := os.Open("data/canada/carsownr.txt.bz2")
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't open Canada owners file")
		return err
	}
	defer f.Close()

	rdr := csv.NewReader(bzip2.NewReader(f))

	txn, err := db.Begin()
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't open transaction")
		return err
	}

	stmt, err := txn.Prepare(pq.CopyIn("canada_owner", "mark", "owner", "city", "province", "country"))
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't prepare Canada owners insert statement")
		return err
	}

	rowCount := 0

	for {
		row, err := rdr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error().Err(err).Msgf("Error reading Canada owner file")
			stmt.Close()
			txn.Rollback()
			return err
		}

		rowCount++

		mark := row[0]
		owner := strings.TrimSpace(row[1])
		city := strings.TrimSpace(row[5])
		province := strings.TrimSpace(row[6])
		country := strings.TrimSpace(row[9])

		_, err = stmt.Exec(mark, owner, city, province, country)
		if err != nil {
			log.Error().Err(err).Msgf("Couldn't insert %s into canada_owner", mark)
			continue
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

	log.Info().Msgf("Inserted %d Canada owners", rowCount)

	return nil
}

func loadCanadaRegistrations(db *sqlx.DB) error {
	f, err := os.Open("data/canada/carscurr.txt.bz2")
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't open Canada registration file")
		return err
	}
	defer f.Close()

	rdr := csv.NewReader(bzip2.NewReader(f))

	txn, err := db.Begin()
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't open transaction")
		return err
	}

	stmt, err := txn.Prepare(pq.CopyIn("canada_reg", "mark", "mfg", "model", "year", "icao", "marktrim"))
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't prepare Canada registration insert statement")
		return err
	}

	rowCount := 0

	for {
		row, err := rdr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error().Err(err).Msgf("Error reading Canada registration file")
			stmt.Close()
			txn.Rollback()
			return err
		}

		rowCount++

		mark := row[0]
		mfg := strings.TrimSpace(row[3])
		model := strings.TrimSpace(row[4])
		yearstr := strings.TrimSpace(row[31])
		icaobin := strings.TrimSpace(row[42])
		marktrim := row[46]

		icao, err := strconv.ParseInt(icaobin, 2, 16)
		if err != nil {
			log.Error().Err(err).Msgf("Couldn't parse Mode S binary for %s", mark)
			continue
		}
		icaohex := fmt.Sprintf("%x", icao)

		year, err := strconv.Atoi(yearstr[0:4])
		if err != nil {
			log.Error().Err(err).Msgf("Couldn't parse year from %s", yearstr)
			continue
		}

		_, err = stmt.Exec(mark, mfg, model, year, icaohex, marktrim)
		if err != nil {
			log.Error().Err(err).Msgf("Couldn't insert %s into canada_reg", mark)
			continue
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

	log.Info().Msgf("Inserted %d Canada owners", rowCount)

	return nil
}

func mergeCanadaData(db *sqlx.DB) error {
	stmt := `
		INSERT INTO registration
		(icao, registration, mfg, model, year, owner, city, state, country, source)
		SELECT
			r.icao, 'C-' || r.marktrim, r.mfg, r.model, r.year, o.owner, o.city, o.province, o.country, 'Canada'
		FROM canada_reg r
		INNER JOIN canada_owner o ON r.mark = o.mark`
	log.Info().Msgf("About to merge Canada data into registration table")
	result, err := db.Exec(stmt)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't insert Canada data")
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil {
		log.Info().Msgf("Inserted %d Canada registration records", rows)
	}
	return nil
}
