package services

import (
	"envdash/internal/structs"
	"envdash/internal/utils"
	"sort"
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

func TestValidationSuccess(t *testing.T) {
	registration := &structs.RegisterCountry{
		Name:    "Canada",
		IsoCode: "ca",
	}
	err := utils.Validation(registration) // Changed from validation() to utils.Validation()
	assert.NoError(t, err)
	assert.Equal(t, "CA", registration.IsoCode)
}

func TestValidationMissingName(t *testing.T) {
	registration := &structs.RegisterCountry{
		IsoCode: "ca",
	}
	err := utils.Validation(registration) // Changed
	assert.Error(t, err)
	assert.Equal(t, "missing required field: name", err.Error())
}

func TestValidationMissingIsoCode(t *testing.T) {
	registration := &structs.RegisterCountry{
		Name: "Canada",
	}
	err := utils.Validation(registration) // Changed
	assert.Error(t, err)
	assert.Equal(t, "missing required field: isoCode", err.Error())
}

func TestValidationInvalidIsoCodeLength(t *testing.T) {
	registration := &structs.RegisterCountry{
		Name:    "Canada",
		IsoCode: "can",
	}
	err := utils.Validation(registration) // Changed
	assert.Error(t, err)
	assert.Equal(t, "isoCode must be 2 letters", err.Error()) // Changed from "two" to "2"
}

func TestValidationIsoCodeTrimmed(t *testing.T) {
	registration := &structs.RegisterCountry{
		Name:    "Canada",
		IsoCode: "  ca  ",
	}
	err := utils.Validation(registration) // Changed
	assert.NoError(t, err)
	assert.Equal(t, "CA", registration.IsoCode)
}

func TestToUpdateFields(t *testing.T) {
	dataUpdate := map[string]any{
		"name":    "Usa", // Valid name
		"isoCode": "us",  // Valid 2-letter code
	}
	updates, err := toUpdateFields(dataUpdate)
    assert.NoError(t, err)
    
    // Sort by Path field alphabetically
	sort.Slice(updates, func(i, j int) bool {
		return updates[i].Path < updates[j].Path
	})
	assert.Len(t, updates, 2)

	assert.Equal(t, "name", updates[1].Path)
	assert.Equal(t, "Usa", updates[1].Value)
}