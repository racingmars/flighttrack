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
	t := &Template{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}
	e.Renderer = t
	e.Use(middleware.Logger())
	e.Use(middleware.Gzip())
	e.GET("/", getFlightsHandler(dao))
	e.GET("/reg", getRegistrationHandler(dao))
	e.GET("/flight/:id", getFlightHandler(dao))
	e.Static("/static", "static")
	e.Logger.Fatal(e.Start(":1324"))
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func getFlightsHandler(dao *data.DAO) func(c echo.Context) error {
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

		var flights []data.Flight
		start := time.Date(userdate.Year(), userdate.Month(), userdate.Day(), 0, 0, 0, 0, time.Local).UTC()
		end := time.Date(start.Year(), start.Month(), start.Day()+1, 0, 0, 0, 0, time.Local).UTC()

		var err error
		// If user wants "active" flights, ignore dates and look for flights that aren't closed
		if submitparam == "Active" {
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
			"DateError":  dateerror,
			"DateString": dateparam,
		}
		return c.Render(http.StatusOK, "flightlist.html", vals)
	}
}

var icaoValidator = regexp.MustCompile(`[0-9a-f]{6}`)

func getRegistrationHandler(dao *data.DAO) func(c echo.Context) error {
	return func(c echo.Context) error {
		icao := c.QueryParam("icao")
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
