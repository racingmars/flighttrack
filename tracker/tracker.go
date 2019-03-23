package tracker

import (
	"time"

	"github.com/racingmars/flighttrack/decoder"
)

const sweepInterval = 30 * time.Second

type FlightHandler interface {
	NewFlight(icaoID string, firstSeen time.Time)
	CloseFlight(icaoID string, lastSeen time.Time, messages int)
	SetIdentity(icaoID, callsign string, change bool)
	AddTrackPoint(icaoID, trackPoint *TrackLog)
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
	Current         TrackLog
	pendingSpeed    bool
	pendingPosition bool
	pendingSquak    bool
	pendingAltitude bool
	evenFrame       *decoder.AdsbPosition
	oddFrame        *decoder.AdsbPosition
}

type TrackLog struct {
	Time         time.Time
	Heading      *int
	VS           *int
	Latitude     *float64
	Longitude    *float64
	Altitude     *int
	AltitudeType *int
	Speed        *int
	SpeedType    *decoder.SpeedType
	Squak        *string
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
	}

	t.sweepIfNeeded(tm)
}

func (t *Tracker) sweepIfNeeded(tm time.Time) {
	if tm.After(t.nextSweep) {
		t.sweep(tm)
	}
}

func (t *Tracker) sweep(tm time.Time) {
	cutoff := time.Now().Add(-60 * time.Second)
	for id := range t.flights {
		if t.flights[id].LastSeen.Before(cutoff) {
			// it's been too long since we've seen this flight
			t.handlers.CloseFlight(id, t.flights[id].LastSeen, t.flights[id].MessageCount)
			delete(t.flights, id)
		}
	}
	t.nextSweep = tm.Add(sweepInterval)
}
