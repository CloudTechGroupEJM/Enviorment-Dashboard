package openaq

import (
	"encoding/json"
	"envdash/internal/structs"
	"testing"
)

func TestMeanAq(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		want   float64
	}{
		{name: "average values", values: []float64{10, 20, 30}, want: 20},
		{name: "empty slice", values: []float64{}, want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := meanAq(tt.values)
			if got != tt.want {
				t.Fatalf("meanAq() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEpaLevel_Boundaries(t *testing.T) {
	tests := []struct {
		pm25 float64
		want string
	}{
		{pm25: -0.1, want: "unknown"},
		{pm25: 12.0, want: "Good"},
		{pm25: 35.4, want: "Moderate"},
		{pm25: 55.4, want: "Unhealthy for Sensitive Groups"},
		{pm25: 150.4, want: "Unhealthy"},
		{pm25: 250.4, want: "Very Unhealthy"},
		{pm25: 300.0, want: "Hazardous"},
	}

	for _, tt := range tests {
		if got := epaLevel(tt.pm25); got != tt.want {
			t.Fatalf("epaLevel(%v) = %q, want %q", tt.pm25, got, tt.want)
		}
	}
}

func TestBuildSensorMap_FiltersOnlyPM(t *testing.T) {
	payload := []byte(`{"results":[{"id":1,"sensors":[{"id":11,"parameter":{"name":"pm25"}},{"id":12,"parameter":{"name":"pm10"}},{"id":13,"parameter":{"name":"o3"}}]}]}`)

	var in struct {
		Results []struct {
			ID      int `json:"id"`
			Sensors []struct {
				ID        int `json:"id"`
				Parameter struct {
					Name string `json:"name"`
				} `json:"parameter"`
			} `json:"sensors"`
		} `json:"results"`
	}

	// Decode into the same shape used by the service helper.
	if err := json.Unmarshal(payload, &in); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	locations := &structsAlias{Results: in.Results}
	got := buildSensorMap(locations.toAirQualityIncoming())

	if len(got) != 2 {
		t.Fatalf("expected 2 PM sensors, got %d", len(got))
	}
	if got[11] != "pm25" {
		t.Fatalf("expected sensor 11 to map to pm25, got %q", got[11])
	}
	if got[12] != "pm10" {
		t.Fatalf("expected sensor 12 to map to pm10, got %q", got[12])
	}
	if _, exists := got[13]; exists {
		t.Fatalf("expected non-PM sensor to be excluded")
	}
}

// structsAlias avoids repeating long anonymous literal syntax from the production struct.
type structsAlias struct {
	Results []struct {
		ID      int `json:"id"`
		Sensors []struct {
			ID        int `json:"id"`
			Parameter struct {
				Name string `json:"name"`
			} `json:"parameter"`
		} `json:"sensors"`
	} `json:"results"`
}

func (s *structsAlias) toAirQualityIncoming() *structs.AirQualityIncoming {
	out := &structs.AirQualityIncoming{}
	out.Results = s.Results
	return out
}
