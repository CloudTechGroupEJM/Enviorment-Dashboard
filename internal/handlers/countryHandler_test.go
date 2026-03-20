package handlers

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

type readCloser struct {
	*strings.Reader
}

func (readCloser) Close() error {
	return nil
}

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func stubDefaultTransport(t *testing.T, fn roundTripFunc) {
	t.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = fn
	t.Cleanup(func() {
		http.DefaultTransport = original
	})
}

func TestFetchCountryData_Success(t *testing.T) {
	stubDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://restcountries.com/v3.1/alpha/NO" {
			t.Fatalf("unexpected request url: %s", req.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       readCloser{strings.NewReader(`[{"name":{"common":"Norway"},"isoCode":"NO","capital":["Oslo"],"latlng":[62.0,10.0],"population":5400000,"area":385207,"currencies":{"NOK":{}}}]`)},
			Header:     make(http.Header),
		}, nil
	})

	got, err := FetchCountryData("NO")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 country, got: %d", len(got))
	}
	if got[0].Name.Common != "Norway" {
		t.Fatalf("expected country name Norway, got: %s", got[0].Name.Common)
	}
	if got[0].IsoCode != "NO" {
		t.Fatalf("expected iso code NO, got: %s", got[0].IsoCode)
	}
	if _, ok := got[0].Currencies["NOK"]; !ok {
		t.Fatalf("expected currencies to contain NOK")
	}
}

func TestFetchCountryData_RequestError(t *testing.T) {
	stubDefaultTransport(t, func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("network down")
	})

	_, err := FetchCountryData("NO")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "error fetching country info") {
		t.Fatalf("expected wrapped fetch error, got: %v", err)
	}
}

func TestFetchCountryData_InvalidJSON(t *testing.T) {
	stubDefaultTransport(t, func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       readCloser{strings.NewReader(`{"bad-json"`)},
			Header:     make(http.Header),
		}, nil
	})

	_, err := FetchCountryData("NO")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "error parsing country info") {
		t.Fatalf("expected wrapped parse error, got: %v", err)
	}
}

func TestFetchCountryData_EmptyArray(t *testing.T) {
	stubDefaultTransport(t, func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       readCloser{strings.NewReader(`[]`)},
			Header:     make(http.Header),
		}, nil
	})

	got, err := FetchCountryData("XX")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 countries, got: %d", len(got))
	}
}
