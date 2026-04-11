package metro

import (
	"math"
	"testing"
)

func TestCalculateMean(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		want   float64
		nan    bool
	}{
		{name: "normal values", values: []float64{10, 20, 30}, want: 20},
		{name: "decimal values", values: []float64{1.5, 2.5}, want: 2},
		{name: "empty slice returns NaN", values: []float64{}, nan: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateMean(tt.values)
			if tt.nan {
				if !math.IsNaN(got) {
					t.Fatalf("calculateMean() = %v, want NaN", got)
				}
				return
			}
			if got != tt.want {
				t.Fatalf("calculateMean() = %v, want %v", got, tt.want)
			}
		})
	}
}
