package decoder

type AdsbIdentification struct {
	TC       int
	EC       int
	Callsign string
}

// SpeedType holds the type of reported speed
type SpeedType int

const (
	// SpeedGS is the groundspeed of the aircraft
	SpeedGS SpeedType = iota

	// SpeedIAS is the indicated airspeed of the aircraft
	SpeedIAS

	// SpeedTAS is the true airspeed of the aircraft
	SpeedTAS
)

type AdsbVelocity struct {
	TC               int
	ST               int
	IntentChange     bool
	SpeedType        SpeedType
	Speed            int
	HeadingAvailable bool
	Heading          int
	VerticalRate     int
}

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
