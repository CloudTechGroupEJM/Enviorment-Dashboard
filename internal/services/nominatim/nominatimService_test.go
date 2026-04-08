package nominatim

import "testing"

func TestGetCapitalCords_EmptyCapital(t *testing.T) {
	service := NewNomService()

	_, err := service.GetCapitalCords("")
	if err == nil {
		t.Fatalf("expected error for empty capital, got nil")
	}
	if err.Error() != "capital city name is empty" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRoundTwoDeci(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{name: "round down", in: 10.124, want: 10.12},
		{name: "round up", in: 10.125, want: 10.13},
		{name: "negative", in: -7.456, want: -7.46},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := roundTwoDeci(tt.in)
			if got != tt.want {
				t.Fatalf("roundTwoDeci(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
