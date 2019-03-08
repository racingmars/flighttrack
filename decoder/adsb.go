package decoder

import (
	"encoding/binary"
)

var adsbTypes = []string{
	"Aircraft identification",
	"Aircraft identification",
	"Aircraft identification",
	"Aircraft identification",
	"Surface position",
	"Surface position",
	"Surface position",
	"Surface position",
	"Airborne position (w/ Baro Altitude)",
	"Airborne position (w/ Baro Altitude)",
	"Airborne position (w/ Baro Altitude)",
	"Airborne position (w/ Baro Altitude)",
	"Airborne position (w/ Baro Altitude)",
	"Airborne position (w/ Baro Altitude)",
	"Airborne position (w/ Baro Altitude)",
	"Airborne position (w/ Baro Altitude)",
	"Airborne position (w/ Baro Altitude)",
	"Airborne position (w/ Baro Altitude)",
	"Airborne velocities",
	"Airborne position (w/ GNSS Height)",
	"Airborne position (w/ GNSS Height)",
	"Airborne position (w/ GNSS Height)",
	"Reserved",
	"Reserved",
	"Reserved",
	"Reserved",
	"Reserved",
	"Aircraft status",
	"Target state and status information",
	"UNKNOWN",
	"Aircraft operation status",
}

func getADSBType(b byte) (int, string) {
	tc, _ := binary.Uvarint([]byte{(b & 0xF8) >> 3})
	typeStr := "UNKNOWN"
	if int(tc-1) < len(adsbTypes) {
		typeStr = adsbTypes[tc-1]
	}
	return int(tc), typeStr
}
