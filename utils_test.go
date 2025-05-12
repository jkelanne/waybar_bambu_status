package main

import (
	"testing"
)

func TestConvertTime(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{90, "1h 30m"},
		{45, "45m"},
		{float64(60), "1h"},
		{float64(125.5), "2h 6m"},
	}

	for _, test := range tests {
		result, err := ConvertTime(test.input)
		if err != nil {
			t.Errorf("ConvertTime(%v) returned error: %v", test.input, err)
		}
		if result != test.expected {
			t.Errorf("ConvertTime(%v) = %q, want %q", test.input, result, test.expected)
		}
	}
}

func TestTemperatureIcon(t *testing.T) {
	//  -- empty used as default
	//  -- quarter if current temperature < target / 4
	//  -- half if current temperature < target / 2
	//  -- tree quarters full
	//  -- full if current temperature >= target (tolerance?)
	tests := []struct {
		inputCurrent float64
		inputTarget  float64
		expected     string
	}{
		{float64(25), float64(100), ""},
		{float64(50), float64(100), ""},
		{float64(75), float64(100), ""},
		{float64(100), float64(100), ""},
	}

	for _, test := range tests {
		result := TemperatureIcon(test.inputCurrent, test.inputTarget)

		if result != test.expected {
			t.Errorf("TemperatureIcon(%v, %v) = %s, want %s", test.inputCurrent, test.inputTarget, result, test.expected)
		}
	}
}
