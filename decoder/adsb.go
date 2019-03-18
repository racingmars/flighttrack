package decoder

import (
	"encoding/binary"
)

type adsbMessageType string

const (
	msgAircraftID            adsbMessageType = "Aircraft identification"
	msgSurfacePosition       adsbMessageType = "Surface position"
	msgAirbornPosWithBaroAlt adsbMessageType = "Airborne position (w/ Baro Altitude)"
	msgAirbornVelocities     adsbMessageType = "Airborne velocities"
	msgAirbornPosWithGNSSAlt adsbMessageType = "Airborne position (w/ GNSS Height)"
	msgReserved              adsbMessageType = "Reserved"
	msgAircraftStatus        adsbMessageType = "Aircraft status"
	msgTargetStateStatus     adsbMessageType = "Target state and status information"
	msgAircraftOpsStatus     adsbMessageType = "Aircraft operation status"
	msgUnknown               adsbMessageType = "UNKNOWN"
)

var adsbTypes = []adsbMessageType{
	msgAircraftID,
	msgAircraftID,
	msgAircraftID,
	msgAircraftID,
	msgSurfacePosition,
	msgSurfacePosition,
	msgSurfacePosition,
	msgSurfacePosition,
	msgAirbornPosWithBaroAlt,
	msgAirbornPosWithBaroAlt,
	msgAirbornPosWithBaroAlt,
	msgAirbornPosWithBaroAlt,
	msgAirbornPosWithBaroAlt,
	msgAirbornPosWithBaroAlt,
	msgAirbornPosWithBaroAlt,
	msgAirbornPosWithBaroAlt,
	msgAirbornPosWithBaroAlt,
	msgAirbornPosWithBaroAlt,
	msgAirbornVelocities,
	msgAirbornPosWithGNSSAlt,
	msgAirbornPosWithGNSSAlt,
	msgAirbornPosWithGNSSAlt,
	msgReserved,
	msgReserved,
	msgReserved,
	msgReserved,
	msgReserved,
	msgAircraftStatus,
	msgTargetStateStatus,
	msgUnknown,
	msgAircraftOpsStatus,
}

func getADSBType(b byte) (int, adsbMessageType) {
	tc, _ := binary.Uvarint([]byte{(b & 0xF8) >> 3})
	typeStr := msgUnknown
	if int(tc-1) < len(adsbTypes) {
		typeStr = adsbTypes[tc-1]
	}
	return int(tc), typeStr
}
