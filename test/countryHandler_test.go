package handlers_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"envdash/internal/handlers"
	"envdash/internal/structs"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func stubDefaultTransport(t *testing.T, fn roundTripFunc) {
	t.Helper()
	originalTransport := http.DefaultTransport
	http.DefaultTransport = fn
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestGetCountryInfoStructReturnsMappedCountryOnValidResponse(t *testing.T) {
	stubDefaultTransport(t, func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `[
			{
				"name": {"common": "Norway"},
				"isoCode": "NO",
				"capital": ["Oslo"],
				"latlng": [62.0, 10.0],
				"population": 5457127,
				"area": 385207,
				"currencies": {"NOK": {"name": "Norwegian krone"}}
			}
		]`), nil
	})

	got, err := handlers.GetCountryInfoStruct("NO")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := structs.OutgoingCountry{
		Name:    "Norway",
		IsoCode: "NO",
		Capital: "Oslo",
		Coordinates: structs.Coordinates{
			Latitude:  62.0,
			Longitude: 10.0,
		},
		Population:   5457127,
		Area:         385207,
		BaseCurrency: "NOK",
	}

	if got != want {
		t.Fatalf("unexpected country info\nwant: %+v\ngot:  %+v", want, got)
	}
}

func TestGetCountryInfoStructReturnsErrorWhenRequestFails(t *testing.T) {
	stubDefaultTransport(t, func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("network down")
	})

	_, err := handlers.GetCountryInfoStruct("NO")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetCountryInfoStructReturnsErrorWhenResponseBodyIsInvalidJSON(t *testing.T) {
	stubDefaultTransport(t, func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{not-json}`), nil
	})

	_, err := handlers.GetCountryInfoStruct("NO")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetCountryInfoStructReturnsEmptyBaseCurrencyWhenCurrenciesAreMissing(t *testing.T) {
	stubDefaultTransport(t, func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `[
			{
				"name": {"common": "Norway"},
				"isoCode": "NO",
				"capital": ["Oslo"],
				"latlng": [62.0, 10.0],
				"population": 5457127,
				"area": 385207
			}
		]`), nil
	})

	got, err := handlers.GetCountryInfoStruct("NO")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.BaseCurrency != "" {
		t.Fatalf("expected empty base currency, got %q", got.BaseCurrency)
	}
}

func TestGetCountryInfoStructPanicsWhenAPIResponseContainsNoCountries(t *testing.T) {
	stubDefaultTransport(t, func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `[]`), nil
	})

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic, got none")
		}
	}()

	_, _ = handlers.GetCountryInfoStruct("NO")
}

func TestGetCountryInfoStructPanicsWhenCapitalOrCoordinatesAreMissing(t *testing.T) {
	stubDefaultTransport(t, func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `[
			{
				"name": {"common": "Norway"},
				"isoCode": "NO",
				"capital": [],
				"latlng": [62.0],
				"population": 5457127,
				"area": 385207,
				"currencies": {"NOK": {}}
			}
		]`), nil
	})

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic, got none")
		}
	}()

	_, _ = handlers.GetCountryInfoStruct("NO")
}
