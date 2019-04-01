package main

import (
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

func main() {
	db, err := getConnection()
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't connect to DB")
	}
	defer db.Close()

	err = truncate(db)
	if err != nil {
		return
	}

	err = loadFAA(db)
	if err != nil {
		log.Error().Err(err).Msgf("Error loading FAA data")
	}

	err = loadCanada(db)
	if err != nil {
		log.Error().Err(err).Msgf("Error loading Canada data")
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
