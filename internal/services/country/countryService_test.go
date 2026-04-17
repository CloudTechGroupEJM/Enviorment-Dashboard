package country

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCountryEmptyQuery(t *testing.T) {
	service := &CountryInternal{}

	_, err := service.GetCountry(context.Background(), "   ")

	assert.Error(t, err, "GetCountry() expected error for empty query")
	if err != nil {
		assert.EqualError(t, err, "country query is required")
	}
}
