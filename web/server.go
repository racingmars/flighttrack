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
	gotemplate "text/template"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/racingmars/flighttrack/decoder"
	"github.com/racingmars/flighttrack/web/data"
)

func main() {
	db, err := getConnection()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	dao := data.New(db)

	e := echo.New()

	t := &template{
		templates: gotemplate.Must(gotemplate.ParseGlob("templates/*.html")),
	}
	e.Renderer = t

	e.Use(middleware.Logger())
	e.Use(middleware.Gzip())

	e.GET("/", getRedirectHandler(http.StatusTemporaryRedirect, "/flights/today"))
	e.GET("/flights", getRedirectHandler(http.StatusTemporaryRedirect, "/flights/today"))
	e.GET("/flights/:when", getFlightsHandler(dao))
	e.GET("/reg/:icao", getRegistrationHandler(dao))
	e.GET("/flight/:id", getFlightHandler(dao))

	e.Static("/static", "static")

	e.Logger.Fatal(e.Start(":1324"))
}

type template struct {
	templates *gotemplate.Template
}

func (t *template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func getRedirectHandler(code int, url string) func(c echo.Context) error {
	return func(c echo.Context) error {
		return c.Redirect(code, url)
	}
}

var validWhen = regexp.MustCompile(`^(today|active|[0-9]{8})$`)

func getFlightsHandler(dao *data.DAO) func(c echo.Context) error {
	return func(c echo.Context) error {
		when := c.Param("when")
		if !validWhen.MatchString(when) {
			return echo.NewHTTPError(http.StatusNotFound, "Unknown flight date format")
		}

		var userdate time.Time
		var err error

		if when == "today" || when == "active" {
			userdate = time.Now()
		} else {
			userdate, err = time.Parse("20060102", when)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid date format (must be YYYYMMDD)")
			}
		}

		visibledate := userdate.Format("2006-01-02")

		var flights []data.Flight
		start := time.Date(userdate.Year(), userdate.Month(), userdate.Day(), 0, 0, 0, 0, time.Local).UTC()
		end := time.Date(start.Year(), start.Month(), start.Day()+1, 0, 0, 0, 0, time.Local).UTC()

		// If user wants "active" flights, ignore dates and look for flights that aren't closed
		if when == "active" {
			flights, err = dao.GetFlightsActive()
		} else {
			flights, err = dao.GetFlightsForDateRange(start, end)
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

		vals := map[string]interface{}{
			"Title":      "Flights",
			"Flights":    flights,
			"DateString": visibledate,
		}
		return c.Render(http.StatusOK, "flightlist.html", vals)
	}
}

var icaoValidator = regexp.MustCompile(`[0-9a-f]{6}`)

func getRegistrationHandler(dao *data.DAO) func(c echo.Context) error {
	return func(c echo.Context) error {
		icao := c.Param("icao")
		icao = strings.ToLower(icao)

		if !icaoValidator.MatchString(icao) {
			return c.String(http.StatusNotAcceptable, "Invalid ICAO/transponder ID (must be hex format)")
		}

		foundRegInfo := false
		regInfo, err := dao.GetRegistration(icao)
		if err == sql.ErrNoRows {
			foundRegInfo = false
		} else if err != nil {
			c.Logger().Error(err, icao)
			return c.String(http.StatusInternalServerError, "Unable to execute database query")
		} else {
			foundRegInfo = true
		}

		flights, err := dao.GetFlightsForAirframe(icao)
		if err != nil {
			c.Logger().Error(err)
			return err
		}

		vals := map[string]interface{}{
			"Title":        "Aircraft Registration Information",
			"FoundRegInfo": foundRegInfo,
			"RegInfo":      regInfo,
			"Flights":      flights,
		}
		return c.Render(http.StatusOK, "registration.html", vals)
	}
}

func getFlightHandler(dao *data.DAO) func(c echo.Context) error {
	return func(c echo.Context) error {
		idstring := c.Param("id")
		id, err := strconv.Atoi(idstring)
		if err != nil {
			return err
		}

		flight, err := dao.GetFlight(id)
		if err != nil {
			return err
		}

		tracklog, err := dao.GetTrackLog(id)
		if err != nil {
			return err
		}

		var hasPosition, hasTrack bool
		var pointLat, pointLon float64

		for _, t := range tracklog {
			if t.Latitude.Valid && hasPosition {
				// If we've seen a previous position, we are now just looking for a change
				// in position.
				if t.Latitude.Float64 != pointLat || t.Longitude.Float64 != pointLon {
					// Yep, a different position. We can draw a track line
					hasTrack = true
					break
				}
			} else if t.Latitude.Valid {
				// First time we've seen a position
				hasPosition = true
				// Save this point as the "point" position if we can't draw a line
				pointLat = t.Latitude.Float64
				pointLon = t.Longitude.Float64
			}
		}

		vals := map[string]interface{}{
			"Title":       "Flight Info",
			"Flight":      flight,
			"TrackLog":    tracklog,
			"HasPosition": hasPosition,
			"HasTrack":    hasTrack,
			"PointLat":    pointLat,
			"PointLon":    pointLon,
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
