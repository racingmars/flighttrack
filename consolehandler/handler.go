package consolehandler

import (
	"fmt"
	"time"

	"github.com/racingmars/flighttrack/tracker"
)

type ConsoleHandler struct {
	callsigns map[string]string
}

func (h *ConsoleHandler) NewFlight(icaoID string, firstSeen time.Time) {
	fmt.Printf("%8s: New flight created.\n", icaoID)
}

func (h *ConsoleHandler) CloseFlight(icaoID string, lastSeen time.Time, messages int) {
	fmt.Printf("%8s: Closed after %d messages\n", h.bestID(icaoID), messages)
	delete(h.callsigns, icaoID)
}

func (h *ConsoleHandler) SetIdentity(icaoID, callsign string, change bool) {
	if h.callsigns == nil {
		h.callsigns = make(map[string]string)
	}

	h.callsigns[icaoID] = callsign
	fmt.Printf("%8s: New callsign for %s", callsign, icaoID)
	if change {
		fmt.Printf("*** CHANGE ***")
	}
	fmt.Printf("\n")
}

func (h *ConsoleHandler) AddTrackPoint(icaoID string, trackPoint tracker.TrackLog) {
	fmt.Printf("%8s:", h.bestID(icaoID))
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

func (h *ConsoleHandler) bestID(icao string) string {
	if h.callsigns == nil {
		h.callsigns = make(map[string]string)
	}

	if callsign, ok := h.callsigns[icao]; ok {
		return callsign
	}
	return icao
}
