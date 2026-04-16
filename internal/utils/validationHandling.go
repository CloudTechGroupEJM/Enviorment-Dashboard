package utils

import (
	"envdash/internal/structs"
	"errors"
	"fmt"
	"log"
	"strings"
	"unicode"
)

// ======================== Registration Validations ========================

// Validation validates and normalizes a complete RegisterCountry payload.
// Parameters:
//   - registration: Pointer to RegisterCountry struct to validate (fields will be modified in place)
//
// Returns:
//   - error: Validation error if any field is invalid, nil if all fields pass validation
func Validation(registration *structs.RegisterCountry) error {
	var err error

	if registration.Name, err = validateName(registration.Name); err != nil {
		return err
	}
	if registration.IsoCode, err = validateIsoCode(registration.IsoCode, "isoCode", 2); err != nil {
		return err
	}

	if registration.Features.TargetCurrencies, err = validateTargetCurrencies(registration.Features.TargetCurrencies); err != nil {
		return err
	}
	return nil
}

// ValidateName validates and normalizes the name field.
// Parameters:
//   - name: The name string to validate
//
// Returns:
//   - string: name
//   - error: Error if name is empty or contains non-letter characters
func validateName(name string) (string, error) {
	log.Println("test reached here")
	name, nameErr := validString(name, "name")
	if nameErr != nil {
		return "", nameErr
	}
	return name, nil
}

// ValidateIsoCode validates and normalizes an ISO code field.
// Parameters:
//   - value: The ISO code string to validate
//   - field: Field name for error messages
//   - length: Expected length of the ISO code (e.g., 2 for country codes, 3 for currency codes)
//
// Returns:
//   - string: Normalized ISO code in uppercase on success
//   - error: Error if code is invalid, wrong length, or contains non-letter characters
func validateIsoCode(value string, field string, length int) (string, error) {
	isoCode, isoCodeErr := validString(value, field)
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
//
// Returns:
//   - string: Trimmed and validated string on success
//   - error: Error if string is empty after trimming or contains non-letter characters
func validString(input string, field string) (string, error) {
	if input == "" {
		return "", errors.New("missing required field: " + field)
	}
	for _, letter := range input {
		if !unicode.IsLetter(letter) && !unicode.IsSpace(letter) {
				return "", errors.New(field + " must contain letters and spaces only")
		}
	}
	return input, nil
}

// ValidateTargetCurrencies validates and normalizes a list of currency ISO codes.
// Parameters:
//   - oldCurrencies: Slice of currency codes to validate
//
// Returns:
//   - []string: Normalized currency codes in uppercase on success
//   - error: Validation error if any currency code is invalid
func validateTargetCurrencies(oldCurrencies []string) ([]string, error) {
	currencies := make([]string, 0, len(oldCurrencies))
	for _, currency := range oldCurrencies {
		var curErr error
		currency, curErr = validateIsoCode(currency, "currency", 3)
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
//
// Returns:
//   - any: Validated value if validation passes
//   - error: Validation error if validation fails
func ValidateFieldValue(key string, value any) (any, error) {
	switch key {
	case "name":
		if strVal, ok := value.(string); ok {
			return validateName(strVal)
		}
	case "isoCode":
		if strVal, ok := value.(string); ok {
			return validateIsoCode(strVal, "isoCode", 2)
		}
	case "targetCurrencies":
		if currencySlice, ok := value.([]string); ok {
			return validateTargetCurrencies(currencySlice)
		}
	}
	return value, nil
}

// ========================= Auth ==============================

// validateEmailFormat checks if email contains required characters: @ and .
//
// Parameters:
//   - emailAddressToValidate: The email address to validate
//
// Returns:
//   - error: Error message if validation fails, nil if valid
func ValidateEmailFormat(emailAddressToValidate string) error {
	trimmedEmail := strings.TrimSpace(emailAddressToValidate)

	if !strings.Contains(trimmedEmail, "@") {
		return errors.New("invalid email format: missing @ symbol")
	}

	if !strings.Contains(trimmedEmail, ".") {
		return errors.New("invalid email format: missing . (dot)")
	}

	atIndex := strings.Index(trimmedEmail, "@")
	dotIndex := strings.LastIndex(trimmedEmail, ".")

	if atIndex >= dotIndex {
		return errors.New("invalid email format: @ must appear before .")
	}
	return nil
}
