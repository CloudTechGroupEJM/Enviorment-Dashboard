package services

import (
    "envdash/internal/structs"
    "testing"

    "cloud.google.com/go/firestore"
    "github.com/stretchr/testify/assert"
)

func TestNewAPIKeyService(testingInstance *testing.T) {
    firestoreClientInstance := &firestore.Client{}
    apiKeyServiceInstance := NewAPIKeyService(firestoreClientInstance)
    assert.NotNil(testingInstance, apiKeyServiceInstance)
    assert.Equal(testingInstance, firestoreClientInstance, apiKeyServiceInstance.client)
}

func TestNewAPIKeyServiceNilClient(testingInstance *testing.T) {
    assert.Panics(testingInstance, func() {
        NewAPIKeyService(nil)
    })
}

func TestGenerateAPIKeyFormat(testingInstance *testing.T) {
    generatedKey := generateAPIKey()
    assert.NotEmpty(testingInstance, generatedKey)
    assert.True(testingInstance, len(generatedKey) > 10, "generated key should be longer than 10 characters")
    assert.Contains(testingInstance, generatedKey, "sk-envdash-", "key should start with sk-envdash-")
}

func TestRegisterNewClientValidation(testingInstance *testing.T) {
    firestoreClientInstance := &firestore.Client{}
    apiKeyServiceInstance := NewAPIKeyService(firestoreClientInstance)

    // Test missing name
    registrationWithoutName := structs.APIKeyRegistration{
        Email: "test@meow.bo",
    }
    _, registrationError := apiKeyServiceInstance.RegisterNewClient(nil, registrationWithoutName)
    assert.NotNil(testingInstance, registrationError)
    assert.Equal(testingInstance, "missing required field: name", registrationError.Error())

    // Test missing email
    registrationWithoutEmail := structs.APIKeyRegistration{
        Name: "test-app",
    }
    _, registrationError = apiKeyServiceInstance.RegisterNewClient(nil, registrationWithoutEmail)
    assert.NotNil(testingInstance, registrationError)
    assert.Equal(testingInstance, "missing required field: email", registrationError.Error())
}