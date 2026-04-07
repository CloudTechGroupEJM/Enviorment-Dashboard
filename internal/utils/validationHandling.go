package utils

import (
	"envdash/internal/structs"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// Validation validates and normalizes a complete RegisterCountry payload.
// Parameters:
//   - registration: Pointer to RegisterCountry struct to validate (fields will be modified in place)
// Returns:
//   - error: Validation error if any field is invalid, nil if all fields pass validation
func Validation(registration *structs.RegisterCountry) error {
    var err error

    if registration.Name, err = ValidateName(registration.Name); err != nil {
        return err
    }
    if registration.IsoCode, err = ValidateIsoCode(registration.IsoCode, "isoCode", 2); err != nil {
        return err
    }

    if registration.Features.TargetCurrencies, err = ValidateTargetCurrencies(registration.Features.TargetCurrencies); err != nil {
        return err
    }
    return nil
}



// ValidateName validates and normalizes the name field.
// Parameters:
//   - name: The name string to validate
// Returns:
//   - string: Trimmed with first letter as upper
//   - error: Error if name is empty or contains non-letter characters
func ValidateName(name string) (string, error) {
	name, nameErr := ValidString(name, "name")
	if nameErr != nil {
		return "", nameErr
	}
	return strings.ToUpper(name[0:1]) + strings.ToLower(name[1:]), nil
}

// ValidateIsoCode validates and normalizes an ISO code field.
// Parameters:
//   - value: The ISO code string to validate
//   - field: Field name for error messages
//   - length: Expected length of the ISO code (e.g., 2 for country codes, 3 for currency codes)
// Returns:
//   - string: Normalized ISO code in uppercase on success
//   - error: Error if code is invalid, wrong length, or contains non-letter characters
func ValidateIsoCode(value, field string, length int) (string, error) {
	isoCode, isoCodeErr := ValidString(value, field)
	if isoCodeErr != nil {
		return "", isoCodeErr
	}
	if len(isoCode) != length {
		return "", fmt.Errorf("%s must be %d letters", field, length)
	}
	return strings.ToUpper(isoCode), nil
}

// ValidString removes whitespace and validates a string field.
// Parameters:
//   - input: The string to validate
//   - field: Field name for error messages
// Returns:
//   - string: Trimmed and validated string on success
//   - error: Error if string is empty after trimming or contains non-letter characters
func ValidString(input, field string) (string, error) {
	input = strings.Join(strings.Fields(input), "")
	if input == "" {
		return "", errors.New("missing required field: " + field)
	}
	for _, letter := range input {
		if !unicode.IsLetter(letter) {
			return "", errors.New(field + " must contain letters only")
		}
	}
	return input, nil
}



// ValidateTargetCurrencies validates and normalizes a list of currency ISO codes.
// Parameters:
//   - oldCurrencies: Slice of currency codes to validate
// Returns:
//   - []string: Normalized currency codes in uppercase on success
//   - error: Validation error if any currency code is invalid
func ValidateTargetCurrencies(oldCurrencies []string) ([]string, error) {
	currencies := make([]string, 0, len(oldCurrencies))
	for _, currency := range oldCurrencies {
		var curErr error
		currency, curErr = ValidateIsoCode(currency, "currency", 3)
		if curErr != nil {
			return nil, curErr
		}
		currencies = append(currencies, strings.ToUpper(currency))
	}
	return currencies, nil
}


// ValidateFieldValue validates a single field based on its key and returns the validated value.
// Parameters:
//   - key: Field name to validate (supported: "name", "isoCode", "targetCurrencies")
//   - value: Value to validate
// Returns:
//   - any: Validated value if validation passes
//   - error: Validation error if validation fails
func ValidateFieldValue(key string, value any) (any, error) {
	switch key {
	case "name":
		if strVal, ok := value.(string); ok {
			return ValidateName(strVal)
		}
	case "isoCode":
		if strVal, ok := value.(string); ok {
			return ValidateIsoCode(strVal, "isoCode", 2)
		}
	case "targetCurrencies":
		if currencySlice, ok := value.([]string); ok {
			return ValidateTargetCurrencies(currencySlice)
		}
	}
	return value, nil
}