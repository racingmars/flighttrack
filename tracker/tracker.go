package tracker

import (
	"math"
	"time"

	"github.com/racingmars/flighttrack/decoder"
)

const sweepInterval = 30 * time.Second
const decayTime = 120 * time.Second
const reportMinInterval = 5 * time.Second

const headingEpsilon = 5
const speedEpsilon = 10
const vsEpsilon = 100
const altitudeEpsilon = 100

type FlightHandler interface {
	NewFlight(icaoID string, firstSeen time.Time)
	CloseFlight(icaoID string, lastSeen time.Time, messages int)
	SetIdentity(icaoID, callsign string, change bool)
	AddTrackPoint(icaoID string, trackPoint TrackLog)
}

type Tracker struct {
	flights   map[string]*flight
	handlers  FlightHandler
	nextSweep time.Time
}

type flight struct {
	IcaoID          string
	FirstSeen       time.Time
	LastSeen        time.Time
	MessageCount    int
	Callsign        *string
	Last            TrackLog
	Current         TrackLog
	pendingSpeed    bool
	pendingPosition bool
	pendingSquak    bool
	pendingAltitude bool
	evenFrame       *decoder.AdsbPosition
	oddFrame        *decoder.AdsbPosition
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
	Squak         string
}

func New(handler FlightHandler) *Tracker {
	t := new(Tracker)
	t.flights = make(map[string]*flight)
	t.handlers = handler
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
		if flt.Callsign == nil {
			flt.Callsign = &v.Callsign
			t.handlers.SetIdentity(icaoID, *flt.Callsign, false)
		}
		if *flt.Callsign != v.Callsign {
			flt.Callsign = &v.Callsign
			t.handlers.SetIdentity(icaoID, *flt.Callsign, true)
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

	if reportable {
		t.report(icaoID, flt, tm)
	}
}

func (t *Tracker) report(icaoID string, flt *flight, tm time.Time) {
	if flt.Last.Time.Add(reportMinInterval).After(flt.Current.Time) {
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
	cutoff := time.Now().Add(-decayTime)
	for id := range t.flights {
		if t.flights[id].LastSeen.Before(cutoff) {
			// it's been too long since we've seen this flight
			t.handlers.CloseFlight(id, t.flights[id].LastSeen, t.flights[id].MessageCount)
			delete(t.flights, id)
		}
	}
	t.nextSweep = tm.Add(sweepInterval)
}
