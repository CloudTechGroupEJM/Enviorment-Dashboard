package dashboard

import (
	"envdash/internal/structs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountryQuery(t *testing.T) {
	tests := []struct {
		name        string
		reg         *structs.RegisterCountry
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name: "iso code preferred when both set",
			reg:  &structs.RegisterCountry{IsoCode: "NO", CountryName: "Norway"},
			want: "NO",
		},
		{
			name: "iso code used when only iso set",
			reg:  &structs.RegisterCountry{IsoCode: "SE"},
			want: "SE",
		},
		{
			name: "country name used when iso missing",
			reg:  &structs.RegisterCountry{CountryName: "Norway"},
			want: "Norway",
		},
		{
			name:        "both empty returns error",
			reg:         &structs.RegisterCountry{},
			wantErr:     true,
			errContains: "neither iso code nor country name",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := countryQuery(tc.reg)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNeedsCoords(t *testing.T) {
	tests := []struct {
		name string
		f    structs.BoolFeature
		want bool
	}{
		{name: "all off", f: structs.BoolFeature{}, want: false},
		{name: "coordinates on", f: structs.BoolFeature{Coordinates: true}, want: true},
		{name: "temperature on", f: structs.BoolFeature{Temperature: true}, want: true},
		{name: "precipitation on", f: structs.BoolFeature{Precipitation: true}, want: true},
		{name: "air quality on", f: structs.BoolFeature{AirQuality: true}, want: true},
		{
			name: "unrelated features don't trigger coords",
			f:    structs.BoolFeature{Population: true, Area: true, Capital: true},
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, needsCoords(tc.f))
		})
	}
}

func TestFirstCurrency(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]struct{}
		want string
	}{
		{name: "empty map returns empty string", in: nil, want: ""},
		{name: "single entry", in: map[string]struct{}{"NOK": {}}, want: "NOK"},
		{
			name: "alphabetically first wins",
			in:   map[string]struct{}{"USD": {}, "EUR": {}, "NOK": {}},
			want: "EUR",
		},
		{
			// Deterministic ordering — run it enough times that a non-sorted
			// impl relying on map iteration would eventually trip this test.
			name: "returns same value across repeated calls",
			in:   map[string]struct{}{"USD": {}, "EUR": {}, "NOK": {}, "SEK": {}, "GBP": {}},
			want: "EUR",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Run multiple times for the determinism case.
			for i := 0; i < 50; i++ {
				assert.Equal(t, tc.want, firstCurrency(tc.in))
			}
		})
	}
}

func TestBuildDashboard(t *testing.T) {
	country := &structs.IncomingCountry{
		IsoCode:    "NO",
		Capital:    []string{"Oslo"},
		Population: 5_379_475,
		Area:       323_802,
	}
	country.Name.Common = "Norway"

	nom := &structs.NomResponse{Lat: 59.9133, Lon: 10.7390}
	metro := &structs.MetroResponse{MeanTemperature: 5.5, MeanPrecipitation: 1.2}

	t.Run("all features on populates everything", func(t *testing.T) {
		req := structs.BoolFeature{
			Coordinates:   true,
			Temperature:   true,
			Precipitation: true,
			Population:    true,
			Area:          true,
			Capital:       true,
		}

		got := buildDashboard(country, nom, metro, nil, nil, req)

		require.NotNil(t, got)
		assert.Equal(t, "Norway", got.Country)
		assert.Equal(t, "NO", got.IsoCode)
		assert.NotEmpty(t, got.LastRetrieval, "LastRetrieval should be set")

		require.NotNil(t, got.Features)
		require.NotNil(t, got.Features.Temperature)
		assert.InDelta(t, 5.5, *got.Features.Temperature, 1e-9)
		require.NotNil(t, got.Features.Precipitation)
		assert.InDelta(t, 1.2, *got.Features.Precipitation, 1e-9)
		require.NotNil(t, got.Features.Coordinates)
		assert.InDelta(t, 59.9133, got.Features.Coordinates.Latitude, 1e-9)
		assert.InDelta(t, 10.7390, got.Features.Coordinates.Longitude, 1e-9)
		assert.Equal(t, "Oslo", got.Features.Capital)
		assert.EqualValues(t, 5_379_475, got.Features.Population)
		assert.EqualValues(t, 323_802, got.Features.Area)
	})

	t.Run("nothing requested leaves features empty", func(t *testing.T) {
		got := buildDashboard(country, nil, nil, nil, nil, structs.BoolFeature{})

		require.NotNil(t, got)
		require.NotNil(t, got.Features)
		assert.Nil(t, got.Features.Temperature)
		assert.Nil(t, got.Features.Precipitation)
		assert.Nil(t, got.Features.Coordinates)
		assert.Empty(t, got.Features.Capital)
	})

	t.Run("coordinates requested but nom nil leaves coords empty", func(t *testing.T) {
		got := buildDashboard(country, nil, nil, nil, nil,
			structs.BoolFeature{Coordinates: true})

		require.NotNil(t, got)
		assert.Nil(t, got.Features.Coordinates,
			"Coordinates must be nil when nominatim data missing, even if requested")
	})

	t.Run("temperature nil when metro missing", func(t *testing.T) {
		got := buildDashboard(country, nom, nil, nil, nil,
			structs.BoolFeature{Temperature: true, Precipitation: true})

		assert.Nil(t, got.Features.Temperature)
		assert.Nil(t, got.Features.Precipitation)
	})

	t.Run("capital requested but country has no capital leaves capital empty", func(t *testing.T) {
		emptyCapital := &structs.IncomingCountry{IsoCode: "NO", Capital: nil}
		emptyCapital.Name.Common = "Norway"

		got := buildDashboard(emptyCapital, nil, nil, nil, nil,
			structs.BoolFeature{Capital: true})

		assert.Empty(t, got.Features.Capital)
	})
}
