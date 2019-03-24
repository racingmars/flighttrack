package consolehandler

import (
	"fmt"
	"time"

	"github.com/racingmars/flighttrack/tracker"
)

type ConsoleHandler struct {
}

func (h ConsoleHandler) NewFlight(icaoID string, firstSeen time.Time) {
	fmt.Println("New flight :", icaoID)
}

func (h ConsoleHandler) CloseFlight(icaoID string, lastSeen time.Time, messages int) {
	fmt.Printf("Closed flight %s after seeing %d messages\n", icaoID, messages)
}

func (h ConsoleHandler) SetIdentity(icaoID, callsign string, change bool) {
	fmt.Printf("New callsign for %s: %s\n", icaoID, callsign)
	if change {
		fmt.Println("*** CHANGE ***")
	}
}

func (h ConsoleHandler) AddTrackPoint(icaoID string, trackPoint tracker.TrackLog) {
	fmt.Printf("New track log for %s.", icaoID)
	if trackPoint.SpeedValid {
		fmt.Printf(" Speed %d.", trackPoint.Speed)
	}
	if trackPoint.HeadingValid {
		fmt.Printf(" Heading: %d.", trackPoint.Heading)
	}
	if trackPoint.VSValid {
		fmt.Printf(" VS: %d.", trackPoint.VS)
	}
	if trackPoint.AltitudeValid {
		fmt.Printf(" Alt: %d.", trackPoint.Altitude)
	}
	fmt.Printf("\n")
}
