package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var skipFAA = flag.Bool("skipfaa", false, "Skip loading FAA data")
var skipCanada = flag.Bool("skipcanada", false, "Skip loading Canada data")
var skipFA = flag.Bool("skipfa", false, "Skip loading Flightaware data")
var noTruncate = flag.Bool("notruncate", false, "Do not truncate registration table before loading")

func main() {
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	db, err := getConnection()
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't connect to DB")
	}
	defer db.Close()

	if !*noTruncate {
		err = truncate(db)
		if err != nil {
			return
		}
	}

	if !*skipFAA {
		err = loadFAA(db)
		if err != nil {
			log.Error().Err(err).Msgf("Error loading FAA data")
		}
	}

	if !*skipCanada {
		err = loadCanada(db)
		if err != nil {
			log.Error().Err(err).Msgf("Error loading Canada data")
		}
	}

	if !*skipFA {
		err = loadFlightaware(db)
		if err != nil {
			log.Error().Err(err).Msgf("Error loading Flightaware data")
		}
	}

}

func getConnection() (*sqlx.DB, error) {
	connStr, ok := os.LookupEnv("DBURL")
	if !ok {
		return nil, fmt.Errorf("DBURL environment variable not set")
	}
	db, err := sqlx.Connect("postgres", connStr)
	return db, err
}

func truncate(db *sqlx.DB) error {
	_, err := db.Exec("TRUNCATE TABLE registration")
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't truncate registration table")
		return err
	}
	return nil
}
