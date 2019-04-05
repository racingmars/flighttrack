package main

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
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
	e.GET("/", getFlightsHandler(db))
	e.GET("/reg", getRegistrationHandler(db))
	e.Logger.Fatal(e.Start(":1324"))
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

type flightsRow struct {
	ID           int            `db:"id"`
	Icao         string         `db:"icao"`
	Callsign     sql.NullString `db:"callsign"`
	FirstSeen    time.Time      `db:"first_seen"`
	LastSeen     pq.NullTime    `db:"last_seen"`
	MsgCount     sql.NullInt64  `db:"msg_count"`
	Registration sql.NullString
	Owner        sql.NullString
	Airline      sql.NullString `db:"airline"`
	TypeCode     sql.NullString `db:"typecode"`
	MfgYear      sql.NullInt64  `db:"year"`
	Mfg          sql.NullString
	Model        sql.NullString
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
		if dateparam != "" && submitparam != "Today" {
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
		err := db.Select(&flights,
			`SELECT f.id, f.icao, f.callsign, f.first_seen, f.last_seen, f.msg_count,
			        r.registration, r.owner, a.name AS airline, r.typecode, r.year, r.mfg, r.model
			 FROM flight f
			 LEFT OUTER JOIN registration r ON f.icao=r.icao
			 LEFT OUTER JOIN airline a ON a.icao=substring(f.callsign from 1 for 3) AND f.icao NOT LIKE 'ae%'
			 WHERE f.first_seen >= $1 AND f.first_seen < $2
			 ORDER BY f.first_seen`,
			start, end)
		if err != nil {
			c.Logger().Error(err)
			return err
		}

		/* for i := range flights {
			if flights[i].Owner.Valid {
				flights[i].Owner.String = strings.Title(strings.ToLower(flights[i].Owner.String))
			}
		} */

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

func getConnection() (*sqlx.DB, error) {
	connStr, ok := os.LookupEnv("DBURL")
	if !ok {
		return nil, fmt.Errorf("DBURL environment variable not set")
	}
	db, err := sqlx.Connect("postgres", connStr)
	return db, err
}
