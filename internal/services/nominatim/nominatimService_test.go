package nominatim

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoundTwoDeci(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  float64
	}{
		{name: "rounds down", value: 12.344, want: 12.34},
		{name: "rounds down (edge)", value: 12.3449, want: 12.34},
		{name: "rounds up", value: 12.345, want: 12.35},
		{name: "already two decimal places", value: 45.67, want: 45.67},
		{name: "one decimal place", value: 45.6, want: 45.6},
		{name: "whole number", value: 45.0, want: 45.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := roundTwoDeci(tc.value)
			assert.Equal(t, tc.want, got, "roundTwoDeci(%v)", tc.value)
		})
	}
}

func TestGetCapitalCoords_EmptyCapital(t *testing.T) {
	service := &NomInternal{}

	_, err := service.GetCapitalCoords(context.Background(), "")

	assert.Error(t, err, "GetCapitalCoords() expected error for empty capital")
	if err != nil {
		assert.EqualError(t, err, "capital city name is empty")
	}
}
