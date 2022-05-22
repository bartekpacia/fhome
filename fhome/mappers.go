package fhome

import (
	"fmt"
	"strconv"
	"strings"
)

var baseLightingValue = 0x6000

// MapLighting maps value to a string that is ready to be passed to Xevent.
//
// Clamps if the value is too small or too big.
func MapLighting(value int) string {
	if value < 0 {
		return "0x6000"
	} else if value > 100 {
		return "0x6064"
	}

	val := baseLightingValue + value
	fval := "0x" + strconv.FormatInt(int64(val), 16)
	return fval
}

func RemapLighting(value string) (int, error) {
	value = strings.TrimPrefix(value, "0x")
	parsed, err := strconv.ParseInt(value, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse int from %s: %v", value, err)
	}

	parsedValue := int(parsed) - baseLightingValue
	return parsedValue, nil
}

var baseTemp float64 = 0xa078 - 12*10.0 // 0°C

// MapTemperature maps val to a string that is ready to be passed to Xevent.
//
// Examples:
//
// 0°C -> 0x6
//
// 12°C -> 0xa078 -> 41080
//
// 25°C -> 0xa0fa -> 41210
//
// 28°C -> 0xa118 -> 41240
func MapTemperature(value float64) string {
	if value < 12 {
		return "0xa078"
	} else if value > 28 {
		return "0xa118"
	}

	v := baseTemp + value*10
	fval := "0x" + strconv.FormatInt(int64(v), 16)
	return fval
}
