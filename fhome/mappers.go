package fhome

import (
	"strconv"
)

// MapLightning maps value to a string that is ready to be passed to Xevent.
//
// Clamps if the value is too small or too big.
func MapLightning(value int) string {
	if value < 0 {
		return "0x6000"
	} else if value > 100 {
		return "0x6064"
	}

	val := 0x6000 + value
	fval := "0x" + strconv.FormatInt(int64(val), 16)
	return fval
}

// MapTemperature maps val to a string that is ready to be passed to Xevent.
//
// Examples:
//
// 12°C -> 0xa078
//
// 25°C -> 0xa0fa
//
// 28°C -> 0xa118
func MapTemperature(value float64) string {
	if value < 12 {
		return "0xa078"
	} else if value > 28 {
		return "0xa118"
	}

	v := 0xa078 + value*10
	fval := "0x" + strconv.FormatInt(int64(v), 16)
	return fval
}
