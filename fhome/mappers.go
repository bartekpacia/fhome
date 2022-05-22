package fhome

import (
	"strconv"
	"strings"
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

func RemapLightning(value string) (int, error) {
	valStr := strings.TrimSuffix(value, "%")

	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, err
	}

	return val, nil
}

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
	base := 0xa078 - 12*10.0 // minimum value

	if value < 12 {
		return "0xa078"
	} else if value > 28 {
		return "0xa118"
	}

	v := base + value*10
	fval := "0x" + strconv.FormatInt(int64(v), 16)
	return fval
}

func RemapValue(value string) {
}
