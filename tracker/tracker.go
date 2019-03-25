package tracker

import (
	"math"
	"time"

	"github.com/racingmars/flighttrack/decoder"

	"github.com/rs/zerolog/log"
)

const sweepInterval = 30 * time.Second
const decayTime = 5 * time.Minute
const reportMinInterval = 5 * time.Second

const headingEpsilon = 5
const speedEpsilon = 10
const vsEpsilon = 100
const altitudeEpsilon = 100
const distanceEpsilonNM = 5

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
	IcaoID       string
	FirstSeen    time.Time
	LastSeen     time.Time
	MessageCount int
	Callsign     *string
	Category     decoder.AircraftType
	Last         TrackLog
	Current      TrackLog
	evenFrame    *decoder.AdsbPosition
	oddFrame     *decoder.AdsbPosition
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
	SquakValid    bool
	Squak         string
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

func (t *Tracker) Message(icaoID string, tm time.Time, msg interface{}) {
	flt, ok := t.flights[icaoID]
	if !ok {
		flt = &flight{IcaoID: icaoID, FirstSeen: tm}
		t.flights[icaoID] = flt
		t.handlers.NewFlight(icaoID, tm)
	}
	flt.LastSeen = tm
	flt.MessageCount++

	switch v := msg.(type) {
	case *decoder.AdsbIdentification:
		flt.Current.IdentityValid = true
		flt.Current.Callsign = v.Callsign
		flt.Current.Category = v.Type
		if flt.Callsign == nil {
			flt.Callsign = &v.Callsign
			flt.Category = v.Type
			t.handlers.SetIdentity(icaoID, *flt.Callsign, flt.Category, false)
		}
		if *flt.Callsign != v.Callsign || flt.Category != v.Type {
			log.Warn().Msgf("Callsign change for %s. Was %s/%d now %s/%d", icaoID, *flt.Callsign, flt.Category, v.Callsign, v.Type)
			flt.Callsign = &v.Callsign
			flt.Category = v.Type
			t.handlers.SetIdentity(icaoID, *flt.Callsign, flt.Category, true)
		}
	case *decoder.AdsbVelocity:
		t.handleAdsbVelocity(icaoID, flt, tm, v)
	case *decoder.AdsbPosition:
		t.handleAdsbPosition(icaoID, flt, tm, v)
	}

	t.sweepIfNeeded(tm)
}

func (t *Tracker) CloseAllFlights() {
	for id := range t.flights {
		t.handlers.CloseFlight(id, t.flights[id].LastSeen, t.flights[id].MessageCount)
		delete(t.flights, id)
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
	}

	if !flt.Current.SpeedValid {
		// This is the first time we've received a speed
		flt.Current.SpeedValid = true
		flt.Current.Speed = msg.Speed
		flt.Current.SpeedType = msg.SpeedType
		reportable = true
	} else {
		flt.Current.Speed = msg.Speed
		flt.Current.SpeedType = msg.SpeedType
		difference := int(math.Abs((float64(flt.Current.Speed - flt.Last.Speed))))
		if difference > speedEpsilon {
			reportable = true
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
	} else if msg.VerticalRateAvailable {
		flt.Current.VS = vs
		difference := int(math.Abs((float64(flt.Current.VS - flt.Last.VS))))
		if difference > vsEpsilon {
			reportable = true
		}
	}

	if reportable {
		t.report(icaoID, flt, tm)
	}
}

func (t *Tracker) handleAdsbPosition(icaoID string, flt *flight, tm time.Time, msg *decoder.AdsbPosition) {
	reportable := false
	flt.Current.Time = tm

	if !flt.Current.AltitudeValid {
		flt.Current.AltitudeValid = true
		reportable = true
	}
	flt.Current.Altitude = msg.Altitude
	difference := int(math.Abs((float64(flt.Current.Altitude - flt.Last.Altitude))))
	if difference > altitudeEpsilon {
		reportable = true
	}

	if msg.Frame == 0 {
		flt.evenFrame = msg
	} else {
		flt.oddFrame = msg
	}
	if flt.evenFrame != nil && flt.oddFrame != nil {
		if lat, lon, good := decoder.CalcPosition(*flt.oddFrame, *flt.evenFrame); good {
			flt.Current.PositionValid = true
			flt.Current.Longitude = lon
			flt.Current.Latitude = lat
			if flt.Current.Longitude != flt.Last.Longitude || flt.Current.Latitude != flt.Last.Latitude {
				reportable = true
			}
			if !flt.Last.PositionValid {
				reportable = true
			} else {
				if math.Abs(distanceNM(lat, flt.Last.Latitude, lon, flt.Last.Longitude)) >= distanceEpsilonNM {
					reportable = true
				}
			}
		}
		// Now that we've used the even and odd frames, discard them to ensure the next calculation
		// is with fresh values.
		flt.evenFrame = nil
		flt.oddFrame = nil
	}

	if reportable {
		t.report(icaoID, flt, tm)
	}
}

func (t *Tracker) report(icaoID string, flt *flight, tm time.Time) {
	if !t.ForceReporting && flt.Last.Time.Add(reportMinInterval).After(flt.Current.Time) {
		// We've too recently sent a previous position report.
		return
	}
	flt.Last = flt.Current
	flt.Last.Time = tm
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
