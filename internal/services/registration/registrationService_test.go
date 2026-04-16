package registration

import (
	"envdash/internal/structs"
	"envdash/internal/utils"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
)

func TestNewRegistrationService(t *testing.T) {
	client := &firestore.Client{}
	service := NewRegistrationService(client)
	assert.NotNil(t, service)
	assert.Equal(t, client, service.client)
}

func TestNewRegistrationServiceNilClient(t *testing.T) {
	assert.Panics(t, func() {
		NewRegistrationService(nil)
	})
}

func TestValidationBothProvided(t *testing.T) {
	registration := &structs.RegisterCountry{
		CountryName: "Canada",
		IsoCode:     "ca",
	}
	err := utils.Validation(registration)
	assert.NoError(t, err)
	assert.Equal(t, "canada", registration.CountryName)
	assert.Equal(t, "CA", registration.IsoCode)
}

func TestValidationOnlyCountryName(t *testing.T) {
	registration := &structs.RegisterCountry{
		CountryName: "Canada",
	}
	err := utils.Validation(registration)
	assert.NoError(t, err)
	assert.Equal(t, "canada", registration.CountryName)
}

func TestValidationOnlyIsoCode(t *testing.T) {
	registration := &structs.RegisterCountry{
		IsoCode: "ca",
	}
	err := utils.Validation(registration)
	assert.NoError(t, err)
	assert.Equal(t, "CA", registration.IsoCode)
}

func TestValidationBothMissing(t *testing.T) {
	registration := &structs.RegisterCountry{}
	err := utils.Validation(registration)
	assert.Error(t, err)
	assert.Equal(t, "at least one of name or isoCode must be provided", err.Error())
}

func TestValidationInvalidIsoCodeLength(t *testing.T) {
	registration := &structs.RegisterCountry{
		IsoCode: "can",
	}
	err := utils.Validation(registration)
	assert.Error(t, err)
	assert.Equal(t, "isoCode must be 2 letters", err.Error())
}

func TestValidationIsoCodeTrimmed(t *testing.T) {
	registration := &structs.RegisterCountry{
		IsoCode: "  ca  ",
	}
	err := utils.Validation(registration)
	assert.NoError(t, err)
	assert.Equal(t, "CA", registration.IsoCode)
}

func TestToUpdateFields(t *testing.T) {
	dataUpdate := map[string]any{
		"name":    "usa", // Valid name
		"isoCode": "us",  // Valid 2-letter code
	}
	updates, err := toUpdateFields(dataUpdate)
	assert.NoError(t, err)

	assert.Len(t, updates, 2)

	assert.Equal(t, "name", updates[1].Path)
	assert.Equal(t, "usa", updates[1].Value)

	assert.Equal(t, "isoCode", updates[0].Path)
	assert.Equal(t, "US", updates[0].Value)
}
