package currency

import (
	"context"
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/structs"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

// CurrencyClient
// holds the currency client
type CurrencyClient struct {
	httpClient *http.Client
}

// NewCurrencyClient
// Creates a new currency client
func NewCurrencyClient() *CurrencyClient {
	return &CurrencyClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// FetchExchangeRates
// Gets the exchange rate information for a given base currency
func (c *CurrencyClient) FetchExchangeRates(ctx context.Context, baseCur string) (*structs.IncomingCurrency, error) {
	u, err := currencyUrl(baseCur)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("building currency request: %w", err)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching exchange rates: %w", err)
	}
	defer res.Body.Close()

	// checks response code
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("currency API returned %d: %s", res.StatusCode, body)
	}

	var rates structs.IncomingCurrency
	if err := json.NewDecoder(res.Body).Decode(&rates); err != nil {
		return nil, fmt.Errorf("decoding currency response: %w", err)
	}

	return &rates, nil
}

// currencyUrl
// builds the exchange rates URL by appending the base currency to the API path
func currencyUrl(baseCur string) (string, error) {
	//check error in base url
	u, err := url.ParseRequestURI(config.CURRENCIES_API)
	if err != nil {
		return "", fmt.Errorf("invalid currency base URL: %w", err)
	}
	u.Path = path.Join(u.Path, baseCur)
	return u.String(), nil
}
