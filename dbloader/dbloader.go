package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
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
var reset = flag.Bool("reset", false, "Reset the flights and track log databases, re-process all raw messages")
var resetonly = flag.Bool("resetonly", false, "Reset the flights and track log databases and quit")
var pretty = flag.Bool("pretty", false, "Use pretty log printing")

var timeToQuit = false

func main() {
	flag.Parse()

	if *pretty {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "01-02 15:04:05"})
	}

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

	if *resetonly {
		*reset = true
	}

	if *reset {
		log.Warn().Msg("Resetting database due to command line flag")
		err := resetDatabase(db)
		if err != nil {
			log.Error().Err(err).Msg("Couldn't reset database")
			return
		}
	}

	if *resetonly {
		return
	}

	trackerstate, handlerstate, lastmsgid, err := loadState(db)
	if lastmsgid > 0 && err == sql.ErrNoRows {
		log.Error().Msg("Last message ID > 0, but unable to load all state rows")
		return
	} else if err == sql.ErrNoRows {
		log.Warn().Msg("No state information found; decoding ALL messages")
	} else if err != nil {
		log.Error().Err(err).Msgf("Unable to load state")
		return
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Warn().Msg("Received termination signal")
		timeToQuit = true
	}()

	loadRows(db, trackerstate, handlerstate, lastmsgid)

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
	ID      int64     `db:"id"`
	Message []byte    `db:"message"`
	Time    time.Time `db:"created_at"`
}

func loadRows(db *sqlx.DB, trackerstate, handlerstate []byte, lastRawMessageID int64) {
	var handler *handler
	var track *tracker.Tracker
	var err error

	if lastRawMessageID > 0 {
		// We should have valid state information from the last run
		if handler, err = newHandlerWithState(db, handlerstate); err != nil {
			log.Error().Err(err).Msg("Couldn't load handler with state")
			return
		}
		if track, err = tracker.NewWithState(handler, false, trackerstate); err != nil {
			log.Error().Err(err).Msg("Couldn't load tracker with state")
			return
		}
	} else {
		handler = newHandler(db)
		track = tracker.New(handler, false)
	}

	defer handler.Close()

	var rows *sqlx.Rows

	for {
		rows, err = db.Queryx("SELECT id, message, created_at FROM raw_message WHERE id>$1 ORDER BY id", lastRawMessageID)
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
				track.Message(icao, msg.Time, decoded)
			}
			if batch == 100000 {
				log.Info().Msgf("processed %d messages", total)
				batch = 0
			}
			lastRawMessageID = msg.ID
		}

		if total > 0 {
			log.Info().Msgf("done: processed %d messages, last msgID %d", total, lastRawMessageID)
		}
		if !hadResult {
			handler.Flush()
			trackerstate := track.GetState()
			handler.saveState(trackerstate, lastRawMessageID)
			if timeToQuit {
				break
			}
			// if we exhausted the backlog, wait a bit for new messages.
			time.Sleep(5 * time.Second)
		}
	}
}

func resetDatabase(db *sqlx.DB) error {
	_, err := db.Exec(`TRUNCATE TABLE flight, tracklog RESTART IDENTITY`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`DELETE FROM parameters WHERE name IN ('trackerstate', 'handlerstate', 'lastmsgid')`)
	if err != nil {
		return err
	}
	return nil
}

func loadState(db *sqlx.DB) (trackerstate, handlerstate []byte, lastmsgid int64, err error) {
	err = db.Get(&lastmsgid, `SELECT value_int FROM parameters WHERE name='lastmsgid'`)
	if err != nil {
		return nil, nil, 0, err
	}

	err = db.Get(&trackerstate, `SELECT value_txt FROM parameters WHERE name='trackerstate'`)
	if err != nil {
		return nil, nil, lastmsgid, err
	}

	err = db.Get(&handlerstate, `SELECT value_txt FROM parameters WHERE name='handlerstate'`)
	if err != nil {
		return nil, nil, lastmsgid, err
	}

	return trackerstate, handlerstate, lastmsgid, nil
}
