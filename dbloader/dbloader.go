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
	ID      int       `db:"id"`
	Message []byte    `db:"message"`
	Time    time.Time `db:"created_at"`
}

func loadRows(db *sqlx.DB) {
	handler := newHandler(db)
	defer handler.Close()
	tracker := tracker.New(handler, false)
	defer tracker.CloseAllFlights()

	var lastRawMessageID int

	var rows *sqlx.Rows
	var err error

	for {
		if lastRawMessageID == 0 {
			rows, err = db.Queryx("SELECT id, message, created_at FROM raw_message ORDER BY id")
		} else {
			rows, err = db.Queryx("SELECT id, message, created_at FROM raw_message WHERE id>$1 ORDER BY id", lastRawMessageID)
		}
		if err != nil {
			log.Error().Err(err).Msg("couldn't query raw messages")
			return
		}

		msg := Message{}
		var total, batch int
		hadResult := false

		for rows.Next() {
			hadResult = true
			total++
			batch++
			err = rows.StructScan(&msg)
			if err != nil {
				log.Error().Err(err).Msg("couldn't scan raw messages")
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
			lastRawMessageID = msg.ID
		}

		if total > 0 {
			log.Info().Msgf("finished: processed %d messages", total)
		}
		if !hadResult {
			handler.Flush()
			// if we exhausted the backlog, wait a bit for new messages.
			time.Sleep(5 * time.Second)
		}
	}
}
