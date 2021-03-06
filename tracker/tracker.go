package tracker

import (
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/racingmars/flighttrack/decoder"
	"github.com/rs/zerolog/log"
)

const sweepInterval = 30 * time.Second
const decayTime = 5 * time.Minute
const reportMinInterval = 5 * time.Second

const headingEpsilon = 10
const speedEpsilon = 10
const vsEpsilon = 150
const altitudeEpsilon = 200
const distanceEpsilonNM = 10

type FlightHandler interface {
	NewFlight(icaoID string, firstSeen time.Time)
	CloseFlight(icaoID string, lastSeen time.Time, messages int)
	SetIdentity(icaoID, callsign string, category decoder.AircraftType, change bool)
	AddTrackPoint(icaoID string, trackPoint TrackLog)
}

type Tracker struct {
	ForceReporting bool
	flights        map[string]*flight
	handlers       FlightHandler
	nextSweep      time.Time
}

type flight struct {
	IcaoID        string
	FirstSeen     time.Time
	LastSeen      time.Time
	MessageCount  int
	Callsign      *string
	Category      decoder.AircraftType
	Last          TrackLog
	Current       TrackLog
	EvenFrame     *decoder.AdsbPosition
	OddFrame      *decoder.AdsbPosition
	PendingChange bool
}

type TrackLog struct {
	Time          time.Time
	Heading       int
	HeadingValid  bool
	VS            int
	VSValid       bool
	PositionValid bool
	Latitude      float64
	Longitude     float64
	AltitudeValid bool
	Altitude      int
	AltitudeType  int
	SpeedValid    bool
	Speed         int
	SpeedType     decoder.SpeedType
	SquawkValid   bool
	Squawk        string
	IdentityValid bool
	Callsign      string
	Category      decoder.AircraftType
}

func New(handler FlightHandler, forceReporting bool) *Tracker {
	t := new(Tracker)
	t.flights = make(map[string]*flight)
	t.handlers = handler
	t.ForceReporting = forceReporting
	return t
}

func NewWithState(handler FlightHandler, forceReporting bool, trackerstate []byte) (*Tracker, error) {
	t := new(Tracker)
	var flights map[string]*flight
	err := json.Unmarshal(trackerstate, &flights)
	if err != nil {
		return nil, err
	}
	t.flights = flights
	t.handlers = handler
	t.ForceReporting = forceReporting
	return t, nil
}

func (t *Tracker) Message(icaoID string, tm time.Time, msg interface{}) {
	flt, ok := t.flights[icaoID]
	if !ok {
		flt = &flight{IcaoID: icaoID, FirstSeen: tm}
		t.flights[icaoID] = flt
		t.handlers.NewFlight(icaoID, tm)
	}
	flt.LastSeen = tm
	flt.MessageCount++

	if msg != nil {
		switch v := msg.(type) {
		case *decoder.AdsbIdentification:
			t.handleAdsbIdentification(icaoID, flt, tm, v)
		case *decoder.AdsbVelocity:
			t.handleAdsbVelocity(icaoID, flt, tm, v)
		case *decoder.AdsbPosition:
			t.handleAdsbPosition(icaoID, flt, tm, v)
		}
	}

	t.sweepIfNeeded(tm)
}

func (t *Tracker) CloseAllFlights() {
	for id := range t.flights {
		if t.flights[id].PendingChange {
			t.report(id, t.flights[id], t.flights[id].LastSeen, true)
		}
		t.handlers.CloseFlight(id, t.flights[id].LastSeen, t.flights[id].MessageCount)
		delete(t.flights, id)
	}
}

func (t *Tracker) handleAdsbIdentification(icaoID string, flt *flight, tm time.Time, msg *decoder.AdsbIdentification) {
	// If there are bad characters, ignore.
	if strings.Contains(msg.Callsign, "#") {
		log.Warn().Msgf("For %s, callsign %s/%d is invalid", icaoID, msg.Callsign, msg.Type)
		return
	}

	// If we already have an identification, and the new type is unknown (probably because it's a BDS2,0 message), use
	// the existing type.
	if flt.Current.IdentityValid && flt.Current.Category != decoder.ACTypeUnknown && msg.Type == decoder.ACTypeUnknown {
		msg.Type = flt.Current.Category
	}

	flt.Current.Time = tm
	flt.Current.IdentityValid = true
	flt.Current.Callsign = msg.Callsign
	flt.Current.Category = msg.Type

	// First time we have a callsign for this flight
	if flt.Callsign == nil {
		flt.Callsign = &msg.Callsign
		flt.Category = msg.Type
		t.handlers.SetIdentity(icaoID, *flt.Callsign, flt.Category, false)
		t.report(icaoID, flt, tm, true)
		return
	}

	// First time we have a good category for this flight
	if *flt.Callsign == msg.Callsign && flt.Category == decoder.ACTypeUnknown && msg.Type != decoder.ACTypeUnknown {
		flt.Category = msg.Type
		t.handlers.SetIdentity(icaoID, *flt.Callsign, flt.Category, false)
		t.report(icaoID, flt, tm, true)
		return
	}

	// This is a change in callsign or category
	if *flt.Callsign != msg.Callsign || flt.Category != msg.Type {
		//log.Warn().Msgf("Callsign change for %s. Was %s/%d now %s/%d", icaoID, *flt.Callsign, flt.Category, msg.Callsign, msg.Type)
		flt.Callsign = &msg.Callsign
		flt.Category = msg.Type
		t.handlers.SetIdentity(icaoID, *flt.Callsign, flt.Category, true)
		t.report(icaoID, flt, tm, true)
		return
	}
}

func (t *Tracker) handleAdsbVelocity(icaoID string, flt *flight, tm time.Time, msg *decoder.AdsbVelocity) {
	reportable := false
	flt.Current.Time = tm

	if !flt.Current.HeadingValid && msg.HeadingAvailable {
		// This is the first time we've received a heading
		flt.Current.HeadingValid = true
		flt.Current.Heading = msg.Heading
		reportable = true
		flt.PendingChange = true
	} else if msg.HeadingAvailable {
		flt.Current.Heading = msg.Heading
		// Normalize headings to be +/- 180 degrees
		oldHeading := flt.Last.Heading
		if oldHeading > 180 {
			oldHeading = oldHeading - 360
		}
		newHeading := flt.Current.Heading
		if newHeading > 180 {
			newHeading = newHeading - 360
		}
		difference := int(math.Abs((float64(newHeading - oldHeading))))
		if difference > headingEpsilon {
			reportable = true
		}
		if difference > 0 {
			flt.PendingChange = true
		}
	}

	if !flt.Current.SpeedValid {
		// This is the first time we've received a speed
		flt.Current.SpeedValid = true
		flt.Current.Speed = msg.Speed
		flt.Current.SpeedType = msg.SpeedType
		reportable = true
		flt.PendingChange = true
	} else {
		flt.Current.Speed = msg.Speed
		flt.Current.SpeedType = msg.SpeedType
		difference := int(math.Abs((float64(flt.Current.Speed - flt.Last.Speed))))
		if difference > speedEpsilon {
			reportable = true
		}
		if difference > 0 {
			flt.PendingChange = true
		}
	}

	// Let's consider +/-64 fpm to be noise around 0
	vs := msg.VerticalRate
	if vs <= 64 && vs >= -64 {
		vs = 0
	}
	if !flt.Current.VSValid && msg.VerticalRateAvailable {
		flt.Current.VSValid = true
		flt.Current.VS = vs
		reportable = true
		flt.PendingChange = true
	} else if msg.VerticalRateAvailable {
		flt.Current.VS = vs
		difference := int(math.Abs((float64(flt.Current.VS - flt.Last.VS))))
		if difference > vsEpsilon {
			reportable = true
		}
		if difference > 0 {
			flt.PendingChange = true
		}
	}

	if reportable {
		t.report(icaoID, flt, tm, false)
	}
}

func (t *Tracker) handleAdsbPosition(icaoID string, flt *flight, tm time.Time, msg *decoder.AdsbPosition) {
	reportable := false
	flt.Current.Time = tm

	if !flt.Current.AltitudeValid {
		flt.Current.AltitudeValid = true
		reportable = true
		flt.PendingChange = true
	}
	flt.Current.Altitude = msg.Altitude
	difference := int(math.Abs((float64(flt.Current.Altitude - flt.Last.Altitude))))
	if difference > altitudeEpsilon {
		reportable = true
	}
	if difference > 0 {
		flt.PendingChange = true
	}

	if msg.Frame == 0 {
		flt.EvenFrame = msg
	} else {
		flt.OddFrame = msg
	}
	if flt.EvenFrame != nil && flt.OddFrame != nil {
		timediff := flt.EvenFrame.Timestamp.Sub(flt.OddFrame.Timestamp)
		if timediff < 0 {
			timediff = -timediff
		}
		if timediff < 5*time.Second {
			if lat, lon, good := decoder.CalcPosition(*flt.OddFrame, *flt.EvenFrame); good {
				flt.Current.PositionValid = true
				flt.Current.Longitude = lon
				flt.Current.Latitude = lat
				if flt.Current.Longitude != flt.Last.Longitude || flt.Current.Latitude != flt.Last.Latitude {
					flt.PendingChange = true
				}
				if !flt.Last.PositionValid {
					reportable = true
					flt.PendingChange = true
				} else {
					if math.Abs(distanceNM(lat, flt.Last.Latitude, lon, flt.Last.Longitude)) >= distanceEpsilonNM {
						reportable = true
					}
				}
			}
		}
	}

	if reportable {
		t.report(icaoID, flt, tm, false)
	}
}

func (t *Tracker) report(icaoID string, flt *flight, tm time.Time, force bool) {
	if !force && !t.ForceReporting && flt.Last.Time.Add(reportMinInterval).After(flt.Current.Time) {
		// We've too recently sent a previous position report.
		return
	}
	flt.Last = flt.Current
	//flt.Last.Time = tm
	flt.PendingChange = false
	t.handlers.AddTrackPoint(icaoID, flt.Last)
}

func (t *Tracker) sweepIfNeeded(tm time.Time) {
	if tm.After(t.nextSweep) {
		t.sweep(tm)
	}
}

func (t *Tracker) sweep(tm time.Time) {
	cutoff := tm.Add(-decayTime)
	for id := range t.flights {
		if t.flights[id].LastSeen.Before(cutoff) {
			// it's been too long since we've seen this flight
			t.handlers.CloseFlight(id, t.flights[id].LastSeen, t.flights[id].MessageCount)
			delete(t.flights, id)
		}
	}
	t.nextSweep = tm.Add(sweepInterval)
}

// Haversine distance between two GPS coordinates
// https://janakiev.com/blog/gps-points-distance-python/
func distanceNM(lat1, lon1, lat2, lon2 float64) float64 {
	const r float64 = 6372800 // Earth radius in meters
	phi1 := lat1 * (math.Pi / 180)
	phi2 := lat2 * (math.Pi / 180)
	dphi := (lat2 - lat1) * (math.Pi / 180)
	dlambda := (lon2 - lon1) * (math.Pi / 180)

	a := math.Pow(math.Sin(dphi/2.0), 2) + math.Cos(phi1)*math.Cos(phi2)*math.Pow(math.Sin(dlambda/2), 2)
	meters := 2 * r * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return meters / 1852
}

func (t *Tracker) GetState() []byte {
	data, err := json.Marshal(t.flights)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't marshal flights array")
		return nil
	}
	return data
}
