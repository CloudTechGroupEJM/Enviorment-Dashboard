package currency

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterRates(t *testing.T) {
	tests := []struct {
		name   string
		rates  map[string]float64
		target []string
		want   map[string]float64
	}{
		{
			name:   "matches exact case",
			rates:  map[string]float64{"USD": 1.1, "EUR": 0.9, "NOK": 10.5},
			target: []string{"USD", "EUR"},
			want:   map[string]float64{"USD": 1.1, "EUR": 0.9},
		},
		{
			name:   "matches case insensitively",
			rates:  map[string]float64{"USD": 1.1, "EUR": 0.9},
			target: []string{"usd", "EuR"},
			want:   map[string]float64{"USD": 1.1, "EUR": 0.9},
		},
		{
			name:   "ignores missing targets",
			rates:  map[string]float64{"USD": 1.1},
			target: []string{"USD", "GBP"},
			want:   map[string]float64{"USD": 1.1},
		},
		{
			name:   "empty target returns empty map",
			rates:  map[string]float64{"USD": 1.1},
			target: []string{},
			want:   map[string]float64{},
		},
		{
			name:   "empty rates returns empty map",
			rates:  map[string]float64{},
			target: []string{"USD"},
			want:   map[string]float64{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := filterRates(tc.rates, tc.target)
			assert.Equal(t, tc.want, got, "filterRates()")
		})
	}
}

func TestGetCurrencyBaseCodeConstraint(t *testing.T) {
	service := &CurrencyInternal{}

	_, err := service.GetCurrency(context.Background(), "", []string{"USD"})

	assert.Error(t, err)
	if err != nil {
		assert.EqualError(t, err, "base currency code is required")
	}
}

func TestGetCurrencyEmptyTargets(t *testing.T) {
	service := &CurrencyInternal{}

	response, err := service.GetCurrency(context.Background(), "NOK", []string{})

	assert.NoError(t, err)
	assert.NotNil(t, response)
	if response != nil {
		assert.Empty(t, response.TargetCurrencies)
	}
}
