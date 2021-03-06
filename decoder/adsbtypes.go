package decoder

import "time"

type AircraftType int

const (
	ACTypeNoInfo AircraftType = iota
	ACTypeLight
	ACTypeSmall
	ACTypeLarge
	ACTypeHighVortexLarge
	ACTypeHeavy
	ACTypeHighPerformance
	ACTypeRotocraft
	ACTypeGlider
	ACTypeLighterThanAir
	ACTypeParachutist
	ACTypeUltralight
	ACTypeUAV
	ACTypeSpaceVehicle
	ACTypeSurfaceEmergency
	ACTypeSurfaceService
	ACTypeObstruction
	ACTypeClusterObstacle
	ACTypeLineObstacle
	ACTypeUnknown
)

type AdsbIdentification struct {
	Callsign string
	Type     AircraftType
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
	TC                    int
	ST                    int
	IntentChange          bool
	SpeedType             SpeedType
	Speed                 int
	HeadingAvailable      bool
	Heading               int
	VerticalRateAvailable bool
	VerticalRate          int
}

type AdsbPosition struct {
	Timestamp time.Time
	TC        int
	SS        int
	Altitude  int
	Frame     int
	LatCPR    int
	LonCPR    int
}

type adsbMessageType string

const (
	msgAircraftID            adsbMessageType = "Aircraft ID"
	msgSurfacePosition       adsbMessageType = "Surface pos."
	msgAirbornPosWithBaroAlt adsbMessageType = "Airborn pos. (w/ Baro Alt)"
	msgAirbornVelocities     adsbMessageType = "Airborn vel."
	msgAirbornPosWithGNSSAlt adsbMessageType = "Airborn pos. (w/ GNSS Hgt)"
	msgReserved              adsbMessageType = "Reserved"
	msgAircraftStatus        adsbMessageType = "Aircraft status"
	msgTargetStateStatus     adsbMessageType = "Target state and status info"
	msgAircraftOpsStatus     adsbMessageType = "Aircraft op status"
	msgUnknown               adsbMessageType = "UNKNOWN"
)

var adsbTypes = []adsbMessageType{
	msgUnknown,
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
