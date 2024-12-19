package api

import (
	"fmt"
	"strconv"
	"strings"
)

var baseLightingValue = 0x6000

// MapLighting maps value to a string that is ready to be passed to
// [Client.SendEvent].
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
	parsed, err := strconv.ParseInt(value, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse int from %s: %v", value, err)
	}

	parsedValue := int(parsed) - baseLightingValue
	return parsedValue, nil
}

var baseTemperatureValue float64 = 0xa078 - 12*10.0 // 0°C

// EncodeTemperature encodes value to represent temperature that is ready to be
// passed to [Client.SendEvent].
//
// Examples of the process:
//
// * 12 -> 41080 + 12 * 10 -> 41080 -> "0xa078"
//
// * 25 -> 41080 + 25 * 10 -> 41210 -> "0xa0fa"
//
// * 28 -> 41080 + 28 * 10 -> 41240 -> "0xa118"
func EncodeTemperature(value float64) string {
	if value < 12 {
		return "0xa078"
	} else if value > 28 {
		return "0xa118"
	}

	v := baseTemperatureValue + value*10
	fval := "0x" + strconv.FormatInt(int64(v), 16)
	return fval
}

// DecodeTemperatureValue converts hex string to float64 representing
// temperature in °C.
//
// Examples:
//
// * "0xa005" -> 0.5
//
// * "0xa078" -> 12.0
//
// * "0xa118" -> 28.0
func DecodeTemperatureValue(value string) (float64, error) {
	v := strings.TrimPrefix(value, "0x")
	parsed, err := strconv.ParseInt(v, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse int from %s: %v", value, err)
	}

	parsedValue := (float64(parsed) - baseTemperatureValue) / 10
	return parsedValue, nil
}

// DecodeTemperatureValueStr converts °C in string to float64.
//
// Example:
//
// "24,0°C" -> 24
func DecodeTemperatureValueStr(value string) (float64, error) {
	v := strings.TrimSuffix(value, "°C")
	v = strings.ReplaceAll(v, ",", ".")
	parsed, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse float from %s: %v", value, err)
	}

	return parsed, nil
}
