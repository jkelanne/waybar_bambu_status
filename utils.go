package main

import (
	"fmt"
	"math"
	"strconv"
)

func ConvertTime(time interface{}) (string, error) {
	var totalMinutes int
	switch v := time.(type) {
	case int:
		totalMinutes = v
	case int8, int16, int32, int64:
		totalMinutes, _ = strconv.Atoi(fmt.Sprintf("%d", v))
	case float32:
		totalMinutes = int(math.Round(float64(v)))
	case float64:
		totalMinutes = int(math.Round(v))
	default:
		return "", fmt.Errorf("unsupported type: %T", time)
	}

	h := totalMinutes / 60
	m := totalMinutes % 60

	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh %dm", h, m), nil
	} else if h > 0 {
		return fmt.Sprintf("%dh", h), nil
	} else {
		return fmt.Sprintf("%dm", m), nil
	}
}

func TemperatureIcon(current, target float64) string {
	//  -- empty used as default
	//  -- quarter if current temperature < target / 4
	//  -- half if current temperature < target / 2
	//  -- tree quarters full
	//  -- full if current temperature >= target (tolerance?)

	if target == 0.0 {
		return ""
	}

	if current <= target/4.0 {
		return ""
	} else if current <= target/2.0 {
		return ""
	} else if current <= (target/4.0)*3.0 {
		return ""
	} else if current >= target {
		return ""
	} else {
		return ""
	}
}
