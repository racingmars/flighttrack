package decoder

import (
	"encoding/hex"
	"fmt"
)

func DecodeMessage(msg string) {
	data, err := hex.DecodeString(msg)
	if err != nil {
		return
	}

	df, ca := GetDFCA(data[0])
	fmt.Printf("DF/CA: %d/%d", df, ca)

	if df == 17 || df == 18 {
		icaoid := data[1:4]
		fmt.Printf(", ICAO ID: %s", hex.EncodeToString(icaoid))

		// if CheckCRC(data) {
		// 	fmt.Printf(", parity passed")
		// } else {
		// 	fmt.Printf(", parity FAILED")
		// }

		typeCode, typeStr := getADSBType(data[4])
		fmt.Printf(". ADS-B: %d - %s", typeCode, typeStr)

		if typeStr == msgAircraftID {
			id := getAdsbIdentification(data[4:])
			fmt.Printf("\nIDENT: %s\n", id.Callsign)
		}

		if typeStr == msgAirbornVelocities {
			vel := getAdsbVelocity(data[4:])
			var speedtype string
			switch vel.SpeedType {
			case SpeedGS:
				speedtype = "groundspeed"
			case SpeedIAS:
				speedtype = "indicated airspeed"
			case SpeedTAS:
				speedtype = "true airspeed"
			}
			fmt.Printf("\nHDG: %03d ; VS: %5dfpm ; SPEED: %3dkt (%s)\n", vel.Heading, vel.VerticalRate, vel.Speed, speedtype)
		}

		if typeStr == msgAirbornPosWithBaroAlt {
			pos := getAdsbPosition(data[4:])
			fmt.Printf("\nALT: %dft\n", pos.Altitude)
			fmt.Printf("\nLatCPR: %10d | LonCPR: %10d | Frame: %d\n", pos.LatCPR, pos.LonCPR, pos.Frame)
		}

	} else if df == 20 || df == 21 {
		crc := CalcCRC(data)
		origcrc := data[len(data)-3:]
		icaoid := []byte{crc[0] ^ origcrc[0], crc[1] ^ origcrc[1], crc[2] ^ origcrc[2]}
		fmt.Printf(", ICAO ID: %s", hex.EncodeToString(icaoid))
	}

	fmt.Printf("\n")
}
