package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/racingmars/flighttrack/decoder"
	"github.com/racingmars/flighttrack/tracker"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
var loglevel = flag.String("level", "info", "Log level: debug, info, warn, error")

func main() {
	flag.Parse()

	switch *loglevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Warn().Msgf("Unknown log level `%s`, setting to WARN", *loglevel)
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal().Err(err).Msg("could not create CPU profile")
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal().Err(err).Msg("could not start CPU profile")
		}
		defer pprof.StopCPUProfile()
	}

	db, err := getConnection()
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't connect to DB")
	}
	defer db.Close()

	loadRows(db)

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal().Err(err).Msg("could not create memory profile")
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal().Err(err).Msg("could not write memory profile")
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

type Message struct {
	Message []byte    `db:"message"`
	Time    time.Time `db:"created_at"`
}

func loadRows(db *sqlx.DB) {
	rows, err := db.Queryx("SELECT message, created_at FROM raw_message ORDER BY id")
	if err != nil {
		fmt.Println(err)
		return
	}

	handler := newHandler(db)
	defer handler.Close()
	tracker := tracker.New(handler, false)
	defer tracker.CloseAllFlights()

	msg := Message{}
	var total, batch int

	for rows.Next() {
		total++
		batch++
		err = rows.StructScan(&msg)
		if err != nil {
			fmt.Println(err)
			return
		}
		icao, decoded := decoder.DecodeMessage(msg.Message, msg.Time)
		if icao != "" && icao != "000000" {
			tracker.Message(icao, msg.Time, decoded)
		}
		if batch == 100000 {
			log.Info().Msgf("processed %d messages", total)
			batch = 0
		}
	}

	log.Info().Msgf("processed %d messages", total)
}
