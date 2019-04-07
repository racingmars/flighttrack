package main

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lib/pq"
	"github.com/racingmars/flighttrack/decoder"
	"github.com/racingmars/flighttrack/web/data"
)

func main() {
	db, err := getConnection()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	e := echo.New()
	t := &Template{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}
	e.Renderer = t
	e.Use(middleware.Logger())
	e.Use(middleware.Gzip())
	e.GET("/", getFlightsHandler(db))
	e.GET("/reg", getRegistrationHandler(db))
	e.GET("/flight/:id", getFlightHandler(db))
	e.Static("/static", "static")
	e.Logger.Fatal(e.Start(":1324"))
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

type flightsRow struct {
	ID             int            `db:"id"`
	Icao           string         `db:"icao"`
	Callsign       sql.NullString `db:"callsign"`
	FirstSeen      time.Time      `db:"first_seen"`
	LastSeen       pq.NullTime    `db:"last_seen"`
	MsgCount       sql.NullInt64  `db:"msg_count"`
	Registration   sql.NullString
	Owner          sql.NullString
	Airline        sql.NullString `db:"airline"`
	TypeCode       sql.NullString `db:"typecode"`
	MfgYear        sql.NullInt64  `db:"year"`
	Mfg            sql.NullString
	Model          sql.NullString
	Icon           string
	IconX          int
	IconY          int
	Category       sql.NullInt64 `db:"category"`
	CategoryString string
}

func getFlightsHandler(db *sqlx.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		// Default to today
		userdate := time.Now()

		// Check if we got a date from the query string, but allow the "today" submission to
		// stick to the default (which we just set to today)
		dateparam := c.QueryParam("date")
		submitparam := c.QueryParam("submit")
		var dateerror bool
		if dateparam != "" && submitparam != "Today" && submitparam != "Active" {
			trialdate, err := time.Parse("2006-01-02", dateparam)
			if err != nil {
				dateerror = true
			} else {
				// Everything good; use the user's date
				userdate = trialdate
			}
		} else {
			// For display purposes, pretend like the user provided the default date
			dateparam = userdate.Format("2006-01-02")
		}

		flights := []flightsRow{}
		start := time.Date(userdate.Year(), userdate.Month(), userdate.Day(), 0, 0, 0, 0, time.Local).UTC()
		end := time.Date(start.Year(), start.Month(), start.Day()+1, 0, 0, 0, 0, time.Local).UTC()

		var err error
		// If user wants "active" flights, ignore dates and look for flights that aren't closed
		if submitparam == "Active" {
			err = db.Select(&flights,
				`SELECT f.id, f.icao, f.callsign, f.first_seen, f.last_seen, f.msg_count, f.category,
						r.registration, r.owner, a.name AS airline, r.typecode, r.year, r.mfg, r.model
				 FROM flight f
				 LEFT OUTER JOIN registration r ON f.icao=r.icao
				 LEFT OUTER JOIN airline a ON a.icao=substring(f.callsign from 1 for 3) AND f.icao NOT LIKE 'ae%'
				 WHERE f.last_seen is null
				 ORDER BY f.first_seen`)
		} else {
			err = db.Select(&flights,
				`SELECT f.id, f.icao, f.callsign, f.first_seen, f.last_seen, f.msg_count, f.category,
						r.registration, r.owner, a.name AS airline, r.typecode, r.year, r.mfg, r.model
				 FROM flight f
				 LEFT OUTER JOIN registration r ON f.icao=r.icao
				 LEFT OUTER JOIN airline a ON a.icao=substring(f.callsign from 1 for 3) AND f.icao NOT LIKE 'ae%'
				 WHERE f.first_seen >= $1 AND f.first_seen < $2
				 ORDER BY f.first_seen`,
				start, end)
		}

		if err != nil {
			c.Logger().Error(err)
			return err
		}

		for i := range flights {
			flights[i].Icon = "unknown.svg"
			if flights[i].TypeCode.Valid {
				if icon, ok := typeToIcon[flights[i].TypeCode.String]; ok {
					flights[i].Icon = fmt.Sprintf("%s.svg", icon)
				}
			}

			// Didn't find a type code match; try against ADS-B identification category
			if flights[i].Icon == "unknown.svg" && flights[i].Category.Valid {
				switch decoder.AircraftType(flights[i].Category.Int64) {
				case decoder.ACTypeLight:
					flights[i].Icon = "cessna.svg"
				case decoder.ACTypeSmall:
					flights[i].Icon = "jet_swept.svg"
				case decoder.ACTypeLarge:
					flights[i].Icon = "airliner.svg"
				case decoder.ACTypeHighVortexLarge:
					flights[i].Icon = "airliner.svg"
				case decoder.ACTypeHeavy:
					flights[i].Icon = "heavy_2e.svg"
				case decoder.ACTypeRotocraft:
					flights[i].Icon = "helicopter.svg"
				}
			}

			flights[i].IconX = iconSize[flights[i].Icon][0]
			flights[i].IconY = iconSize[flights[i].Icon][1]
		}

		vals := struct {
			Title      string
			Flights    []flightsRow
			DateError  bool
			DateString string
		}{
			Title:      "Flights",
			Flights:    flights,
			DateError:  dateerror,
			DateString: dateparam,
		}
		return c.Render(http.StatusOK, "flightlist.html", vals)
	}
}

var icaoValidator = regexp.MustCompile(`[0-9a-f]{6}`)

func getRegistrationHandler(db *sqlx.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		icao := c.QueryParam("icao")
		icao = strings.ToLower(icao)

		if !icaoValidator.MatchString(icao) {
			return c.String(http.StatusNotAcceptable, "Invalid ICAO/transponder ID (must be hex format)")
		}

		foundRegInfo := false
		type RegInfo struct {
			Icao, Source                                                    string
			Registration, Typecode, Mfg, Model, Owner, City, State, Country sql.NullString
			Year                                                            sql.NullInt64
		}
		regInfo := RegInfo{}
		if err := db.Get(&regInfo,
			`SELECT icao, registration, typecode, mfg, model, year, owner, city, state, country, source
			FROM registration
			WHERE icao=$1`, icao); err == sql.ErrNoRows {
			foundRegInfo = false
		} else if err != nil {
			c.Logger().Error(err, icao)
			return c.String(http.StatusInternalServerError, "Unable to execute database query")
		} else {
			foundRegInfo = true
		}

		flights := []flightsRow{}
		err := db.Select(&flights,
			`SELECT f.id, f.icao, f.callsign, f.first_seen, f.last_seen, f.msg_count,
			        a.name AS airline
			 FROM flight f
			 LEFT OUTER JOIN airline a ON a.icao=substring(f.callsign from 1 for 3) AND f.icao NOT LIKE 'ae%'
			 WHERE f.icao = $1
			 ORDER BY f.first_seen`,
			icao)
		if err != nil {
			c.Logger().Error(err)
			return err
		}

		vals := struct {
			Title        string
			FoundRegInfo bool
			RegInfo      RegInfo
			Flights      []flightsRow
		}{
			Title:        "Aircraft Registration Information",
			FoundRegInfo: foundRegInfo,
			RegInfo:      regInfo,
			Flights:      flights,
		}
		return c.Render(http.StatusOK, "registration.html", vals)
	}
}

func getFlightHandler(db *sqlx.DB) func(c echo.Context) error {
	d := data.New(db)
	return func(c echo.Context) error {
		idstring := c.Param("id")
		id, err := strconv.Atoi(idstring)
		if err != nil {
			return err
		}

		flight, err := d.GetFlight(id)
		if err != nil {
			return err
		}

		tracklog, err := d.GetTrackLog(id)
		if err != nil {
			return err
		}

		vals := map[string]interface{}{
			"Title":    "Flight Info",
			"Flight":   flight,
			"TrackLog": tracklog,
		}
		return c.Render(http.StatusOK, "flightdetail.html", vals)
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
