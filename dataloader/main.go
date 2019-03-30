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

	err = loadFAA(db)
	if err != nil {
		log.Error().Err(err).Msgf("Error loading FAA data")
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
