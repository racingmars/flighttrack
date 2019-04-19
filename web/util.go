package main

import (
	"fmt"
	"math"
)

func PrettyLat(decimal float64) string {
	return prettyDegrees(decimal, "N", "S")
}

func PrettyLon(decimal float64) string {
	return prettyDegrees(decimal, "E", "W")
}

func prettyDegrees(decimal float64, positive, negative string) string {
	sign := positive
	if decimal < 0 {
		decimal = math.Abs(decimal)
		sign = negative
	}
	degrees := math.Floor(decimal)
	minutes := 60 * (decimal - degrees)
	result := fmt.Sprintf("%s %d° %.3f′", sign, int(degrees), minutes)
	return result
}
