package tracker

import (
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/racingmars/flighttrack/decoder"
)

type handler struct {
}

func (h *handler) NewFlight(icaoID string, firstSeen time.Time)                                    {}
func (h *handler) CloseFlight(icaoID string, lastSeen time.Time, messages int)                     {}
func (h *handler) SetIdentity(icaoID, callsign string, category decoder.AircraftType, change bool) {}
func (h *handler) AddTrackPoint(icaoID string, trackPoint TrackLog) {
	fmt.Printf("%8s:", icaoID)
	if trackPoint.HeadingValid {
		fmt.Printf(" %03d°", trackPoint.Heading)
	} else {
		fmt.Printf("     ")
	}
	if trackPoint.SpeedValid {
		fmt.Printf(" %3dkts", trackPoint.Speed)
	} else {
		fmt.Printf("       ")
	}
	if trackPoint.AltitudeValid {
		fmt.Printf(" %5dft", trackPoint.Altitude)
	} else {
		fmt.Printf("        ")
	}
	if trackPoint.VSValid {
		fmt.Printf(" %5dfpm", trackPoint.VS)
	} else {
		fmt.Printf("         ")
	}
	if trackPoint.PositionValid {
		fmt.Printf(" %f/%f", trackPoint.Latitude, trackPoint.Longitude)
	}
	fmt.Printf("\n")
}

func TestPosition(t *testing.T) {
	h := new(handler)
	tracker := New(h, true)

	msgEven, _ := hex.DecodeString("8D75804B580FF2CF7E9BA6F701D0")
	msgOdd, _ := hex.DecodeString("8D75804B580FF6B283EB7A157117")
	icao, decoded := decoder.DecodeMessage(msgEven, time.Now())
	tracker.Message(icao, time.Now(), decoded)
	icao, decoded = decoder.DecodeMessage(msgOdd, time.Now())
	tracker.Message(icao, time.Now(), decoded)
}

func TestDistance(t *testing.T) {
	distance := distanceNM(51.5073219, -0.1276474, 52.5170365, 13.3888599)
	if !(distance > 502 && distance < 503) {
		t.Errorf("Distance was %f, should be 502.55nm", distance)
	}
}
