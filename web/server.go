package main

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
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
	e.Logger.Fatal(e.Start(":1323"))
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

type FlightsRow struct {
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
}

func getFlightsHandler(db *sqlx.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		flights := []FlightsRow{}
		start := time.Date(2019, time.April, 1, 0, 0, 0, 0, time.Local).UTC()
		end := time.Date(2019, time.April, 2, 0, 0, 0, 0, time.Local).UTC()
		err := db.Select(&flights,
			`SELECT f.id, f.icao, f.callsign, f.first_seen, f.last_seen, f.msg_count,
			        r.registration, r.owner, a.name AS airline, r.typecode
			 FROM flight f
			 LEFT OUTER JOIN registration r ON f.icao=r.icao
			 LEFT OUTER JOIN airline a ON a.icao=substring(f.callsign from 1 for 3)
			 WHERE f.first_seen >= $1 AND f.first_seen < $2
			 ORDER BY f.first_seen`,
			start, end)
		if err != nil {
			c.Logger().Error(err)
			return err
		}

		for i := range flights {
			if flights[i].Owner.Valid {
				flights[i].Owner.String = strings.Title(strings.ToLower(flights[i].Owner.String))
			}
		}

		vals := struct {
			Title   string
			Flights []FlightsRow
		}{
			Title:   "Flights",
			Flights: flights,
		}
		return c.Render(http.StatusOK, "flightlist.html", vals)
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
