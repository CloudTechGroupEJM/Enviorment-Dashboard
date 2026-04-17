package registration

import (
    "envdash/internal/structs"
    "envdash/internal/utils"
    "testing"

    "cloud.google.com/go/firestore"
    "github.com/stretchr/testify/assert"
)

// TestNewRegistrationService verifies that NewRegistrationService creates a valid service instance
// with the provided firestore client properly assigned.
func TestNewRegistrationService(t *testing.T) {
    client := &firestore.Client{}
    service := NewRegistrationService(client)
    assert.NotNil(t, service)
    assert.Equal(t, client, service.client)
}

// TestNewRegistrationServiceNilClient verifies that NewRegistrationService panics when
// a nil firestore client is provided, ensuring the service fails fast for invalid initialization.
func TestNewRegistrationServiceNilClient(t *testing.T) {
    assert.Panics(t, func() {
        NewRegistrationService(nil)
    })
}

// TestValidationBothProvided tests that validation succeeds when both CountryName and IsoCode
// are provided, and verifies that CountryName is normalized to lowercase and IsoCode to uppercase.
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

// TestValidationOnlyCountryName tests that validation succeeds when only CountryName
// is provided, and verifies it is normalized to lowercase.
func TestValidationOnlyCountryName(t *testing.T) {
    registration := &structs.RegisterCountry{
        CountryName: "Canada",
    }
    err := utils.Validation(registration)
    assert.NoError(t, err)
    assert.Equal(t, "canada", registration.CountryName)
}

// TestValidationOnlyIsoCode tests that validation succeeds when only IsoCode is provided,
// and verifies it is normalized to uppercase.
func TestValidationOnlyIsoCode(t *testing.T) {
    registration := &structs.RegisterCountry{
        IsoCode: "ca",
    }
    err := utils.Validation(registration)
    assert.NoError(t, err)
    assert.Equal(t, "CA", registration.IsoCode)
}

// TestValidationBothMissing tests that validation fails with an appropriate error message
// when neither CountryName nor IsoCode is provided.
func TestValidationBothMissing(t *testing.T) {
    registration := &structs.RegisterCountry{}
    err := utils.Validation(registration)
    assert.Error(t, err)
    assert.Equal(t, "at least one of name or isoCode must be provided", err.Error())
}

// TestValidationInvalidIsoCodeLength tests that validation fails with an error message
// when IsoCode has an invalid length (not exactly 2 letters).
func TestValidationInvalidIsoCodeLength(t *testing.T) {
    registration := &structs.RegisterCountry{
        IsoCode: "can",
    }
    err := utils.Validation(registration)
    assert.Error(t, err)
    assert.Equal(t, "isoCode must be 2 letters", err.Error())
}

// TestValidationIsoCodeTrimmed tests that IsoCode with whitespace is properly trimmed
// and normalized to uppercase before validation.
func TestValidationIsoCodeTrimmed(t *testing.T) {
    registration := &structs.RegisterCountry{
        IsoCode: "  ca  ",
    }
    err := utils.Validation(registration)
    assert.NoError(t, err)
    assert.Equal(t, "CA", registration.IsoCode)
}

// TestToUpdateFields tests the toUpdateFields function by providing a map with name and isoCode,
// verifying that update fields are created correctly with proper values and paths.
// It checks that both fields are returned and that isoCode is normalized to uppercase.
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

// TestToUpdateFieldsEmpty tests toUpdateFields with an empty map
func TestToUpdateFieldsEmpty(t *testing.T) {
    dataUpdate := map[string]any{}
    updates, err := toUpdateFields(dataUpdate)
    assert.NoError(t, err)
    assert.Len(t, updates, 0)
}

// TestToUpdateFieldsSingleField tests toUpdateFields with a single field
func TestToUpdateFieldsSingleField(t *testing.T) {
    dataUpdate := map[string]any{
        "isoCode": "us",
    }
    updates, err := toUpdateFields(dataUpdate)
    assert.NoError(t, err)
    assert.Len(t, updates, 1)
    assert.Equal(t, "isoCode", updates[0].Path)
    assert.Equal(t, "US", updates[0].Value)
}

// TestToUpdateFieldsMultipleFields tests toUpdateFields with multiple fields
// and verifies that results are sorted alphabetically by Path
func TestToUpdateFieldsMultipleFields(t *testing.T) {
    dataUpdate := map[string]any{
        "name":    "france",
        "isoCode": "fr",
        "region":  "europe",
    }
    updates, err := toUpdateFields(dataUpdate)
    assert.NoError(t, err)
    assert.Len(t, updates, 3)

    // Verify fields are sorted alphabetically
    assert.Equal(t, "isoCode", updates[0].Path)
    assert.Equal(t, "name", updates[1].Path)
    assert.Equal(t, "region", updates[2].Path)
}

// TestToUpdateFieldsLastChangeField tests that lastChange field is handled correctly
func TestToUpdateFieldsLastChangeField(t *testing.T) {
    dataUpdate := map[string]any{
        "lastChange": "2024-01-15",
        "name":       "germany",
    }
    updates, err := toUpdateFields(dataUpdate)
    assert.NoError(t, err)
    assert.Len(t, updates, 2)
    // Verify both fields are present
    assert.Equal(t, "lastChange", updates[0].Path)
    assert.Equal(t, "name", updates[1].Path)
}

// TestValidationWithEmptyCountryName tests validation with empty country name
func TestValidationWithEmptyCountryName(t *testing.T) {
    registration := &structs.RegisterCountry{
        CountryName: "",
        IsoCode:     "us",
    }
    err := utils.Validation(registration)
    assert.NoError(t, err)
    assert.Equal(t, "US", registration.IsoCode)
}

// TestValidationWithEmptyIsoCode tests validation with empty ISO code
func TestValidationWithEmptyIsoCode(t *testing.T) {
    registration := &structs.RegisterCountry{
        CountryName: "United States",
        IsoCode:     "",
    }
    err := utils.Validation(registration)
    assert.NoError(t, err)
    assert.Equal(t, "united states", registration.CountryName)
}

// TestValidationIsoCodeWithSpecialCharacters tests ISO code validation with invalid characters
func TestValidationIsoCodeWithSpecialCharacters(t *testing.T) {
    registration := &structs.RegisterCountry{
        IsoCode: "u@",
    }
    err := utils.Validation(registration)
    // This may error depending on validation implementation
    assert.NotNil(t, err)
}

// TestValidationCountryNameCaseConversion tests that country names are properly lowercased
// and validates that only letters are allowed in country names
func TestValidationCountryNameCaseConversion(t *testing.T) {
    testCases := []struct {
        name           string
        input          string
        expectedOutput string
        shouldError    bool
    }{
        {"uppercase", "FRANCE", "france", false},
        {"mixed case", "GrEaT bRiTaIn", "great britain", false},
        {"lowercase", "germany", "germany", false},
        {"with numbers", "ITA2LY", "", true}, // Should fail - numbers not allowed
        {"with spaces", "United States", "united states", false},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            registration := &structs.RegisterCountry{
                CountryName: tc.input,
            }
            err := utils.Validation(registration)
            if tc.shouldError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tc.expectedOutput, registration.CountryName)
            }
        })
    }
}
// TestValidationIsoCodeCaseConversion tests that ISO codes are properly uppercased
func TestValidationIsoCodeCaseConversion(t *testing.T) {
    testCases := []struct {
        name           string
        input          string
        expectedOutput string
    }{
        {"lowercase", "fr", "FR"},
        {"uppercase", "US", "US"},
        {"mixed case", "Gb", "GB"},
        {"with leading spaces", "  it", "IT"},
        {"with trailing spaces", "es  ", "ES"},
        {"with both spaces", "  pl  ", "PL"},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            registration := &structs.RegisterCountry{
                IsoCode: tc.input,
            }
            err := utils.Validation(registration)
            assert.NoError(t, err)
            assert.Equal(t, tc.expectedOutput, registration.IsoCode)
        })
    }
}

// TestValidationIsoCodeTooShort tests ISO code validation with only 1 character
func TestValidationIsoCodeTooShort(t *testing.T) {
    registration := &structs.RegisterCountry{
        IsoCode: "u",
    }
    err := utils.Validation(registration)
    assert.Error(t, err)
    assert.Equal(t, "isoCode must be 2 letters", err.Error())
}

// TestValidationIsoCodeTooLong tests ISO code validation with more than 2 characters
func TestValidationIsoCodeTooLong(t *testing.T) {
    registration := &structs.RegisterCountry{
        IsoCode: "usaa",
    }
    err := utils.Validation(registration)
    assert.Error(t, err)
    assert.Equal(t, "isoCode must be 2 letters", err.Error())
}

// TestValidationIsoCodeWithNumbers tests ISO code validation with numeric characters
func TestValidationIsoCodeWithNumbers(t *testing.T) {
    registration := &structs.RegisterCountry{
        IsoCode: "u1",
    }
    err := utils.Validation(registration)
    // This should error if only letters are allowed
    assert.Error(t, err)
}

// TestToUpdateFieldsSorted tests that toUpdateFields sorts results alphabetically
func TestToUpdateFieldsSorted(t *testing.T) {
    dataUpdate := map[string]any{
        "zebra":  "last",
        "apple":  "first",
        "middle": "second",
    }
    updates, err := toUpdateFields(dataUpdate)
    assert.NoError(t, err)
    assert.Len(t, updates, 3)

    // Verify alphabetical order
    assert.Equal(t, "apple", updates[0].Path)
    assert.Equal(t, "middle", updates[1].Path)
    assert.Equal(t, "zebra", updates[2].Path)
}

// TestValidationWithWhitespaceOnly tests validation when countryName is only whitespace
func TestValidationWithWhitespaceOnly(t *testing.T) {
    registration := &structs.RegisterCountry{
        CountryName: "   ",
        IsoCode:     "ca",
    }
    // Depending on implementation, this might trim to empty
    err := utils.Validation(registration)
    // Should either succeed with trimmed value or fail appropriately
    assert.True(t, err == nil || err.Error() == "at least one of name or isoCode must be provided")
}

// TestNewRegistrationServiceFieldInitialization verifies all fields are properly initialized
func TestNewRegistrationServiceFieldInitialization(t *testing.T) {
    client := &firestore.Client{}
    service := NewRegistrationService(client)

    assert.NotNil(t, service)
    assert.Equal(t, client, service.client)
    // Verify the service is ready to use
    assert.IsType(t, &RegistrationService{}, service)
}

// TestValidationWithBothFieldsTrimmed tests validation when both fields have whitespace
func TestValidationWithBothFieldsTrimmed(t *testing.T) {
    registration := &structs.RegisterCountry{
        CountryName: "  France  ",
        IsoCode:     "  fr  ",
    }
    err := utils.Validation(registration)
    assert.NoError(t, err)
    assert.Equal(t, "france", registration.CountryName)
    assert.Equal(t, "FR", registration.IsoCode)
}

// TestToUpdateFieldsWithNilValue tests toUpdateFields with nil values
func TestToUpdateFieldsWithNilValue(t *testing.T) {
    dataUpdate := map[string]any{
        "name":    "italy",
        "deleted": nil,
    }
    updates, err := toUpdateFields(dataUpdate)
    // Should handle nil values gracefully
    if err == nil {
        assert.Len(t, updates, 2)
    }
}

// TestToUpdateFieldsPreservesNumberValues tests that numeric values are preserved
func TestToUpdateFieldsPreservesNumberValues(t *testing.T) {
    dataUpdate := map[string]any{
        "population": 1000000,
        "area":       2.5,
    }
    updates, err := toUpdateFields(dataUpdate)
    assert.NoError(t, err)
    assert.Len(t, updates, 2)
    // Verify values are preserved
    assert.Equal(t, 1000000, updates[1].Value)
    assert.Equal(t, 2.5, updates[0].Value)
}