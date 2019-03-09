package main

// This is a temporary stand-alone utility to log the raw transponder packets
// to a database before the rest of the application is ready to log more
// interesting information.

// Run with the connection string to Postgres in env variable "DBURL"
// And the dump1090 beast host:port in "DUMP1090HOST"
// e.g.
// $ DBURL="user=flights dbname=flights sslmode=disable" \
//   DUMP1090HOST="piaware:30005" \
//   ./dblogger

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	_ "github.com/lib/pq"
	"github.com/racingmars/flighttrack/beast"
)

func main() {
	db, err := getConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	feed, ok := os.LookupEnv("DUMP1090HOST")
	if !ok {
		log.Print("DUMP1090HOST env variable not set")
		return
	}

	feedconn, err := net.Dial("tcp", feed)
	if err != nil {
		log.Print(err)
		return
	}
	defer feedconn.Close()

	rdr := beast.New(feedconn)
	for {
		msg, offset, err := rdr.Read()
		if err == io.EOF {
			break
		}
		if _, ok := err.(beast.UnknownFormatError); ok {
			log.Print(offset, err)
			continue
		}
		if err != nil {
			log.Print(offset, err)
			return
		}
		err = saveMessage(db, msg)
		if err != nil {
			log.Print(err)
		}
	}
}

func getConnection() (*sql.DB, error) {
	connStr, ok := os.LookupEnv("DBURL")
	if !ok {
		return nil, fmt.Errorf("DBURL environment variable not set.")
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func saveMessage(db *sql.DB, msg *beast.Message) error {
	_, err := db.Exec("INSERT INTO raw_message (message, timestamp, signal) VALUES ($1, $2, $3)",
		msg.Message, msg.Timestamp, uint(msg.SignalLevel))
	return err
}
