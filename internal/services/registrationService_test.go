package services

import (
	"envdash/internal/structs"
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
	err := validation(registration)
	assert.NoError(t, err)
	assert.Equal(t, "CA", registration.IsoCode) // Should be uppercase
}

func TestValidationMissingName(t *testing.T) {
	registration := &structs.RegisterCountry{
		IsoCode: "ca",
	}
	err := validation(registration)
	assert.Error(t, err)
	assert.Equal(t, "missing required field: name", err.Error())
}

func TestValidationMissingIsoCode(t *testing.T) {
	registration := &structs.RegisterCountry{
		Name: "Canada",
	}
	err := validation(registration)
	assert.Error(t, err)
	assert.Equal(t, "missing required field: isoCode", err.Error())
}

func TestValidationInvalidIsoCodeLength(t *testing.T) {
	registration := &structs.RegisterCountry{
		Name:    "Canada",
		IsoCode: "can",
	}
	err := validation(registration)
	assert.Error(t, err)
	assert.Equal(t, "isoCode must be two letters", err.Error())
}

func TestValidationIsoCodeTrimmed(t *testing.T) {
	registration := &structs.RegisterCountry{
		Name:    "Canada",
		IsoCode: "  ca  ",
	}
	err := validation(registration)
	assert.NoError(t, err)
	assert.Equal(t, "CA", registration.IsoCode)
}

func TestToUpdateFields(t *testing.T) {
	dataUpdate := &map[string]any{
		"Name":    "USA",
		"IsoCode": "UsS",
	}
	updates := toUpdateFields(dataUpdate)
	assert.Len(t, updates, 2)
	assert.Equal(t, "Name", updates[0].Path)
	assert.Equal(t, "USA", updates[0].Value)
}
