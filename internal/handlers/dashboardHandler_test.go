package handlers

import (
	"context"
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/structs"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDashboardService struct {
	resp *structs.DashboardResponse
	err  error

	calls      int
	lastID     string
	lastAPIKey string
}

func (f *fakeDashboardService) GetDashboard(ctx context.Context, id, apiKey string) (*structs.DashboardResponse, error) {
	f.calls++
	f.lastID = id
	f.lastAPIKey = apiKey
	return f.resp, f.err
}

type fakeDispatcher struct {
	lifecycleCount int
	thresholdCount int
}

func (f *fakeDispatcher) DispatchThresholdAsync(iso, metric string, value float64) {
	f.thresholdCount++
}

func TestDashboardHandler(t *testing.T) {
	okResp := &structs.DashboardResponse{IsoCode: "NO"}

	tests := []struct {
		name             string
		method           string
		id               string
		apiKey           string
		serviceResp      *structs.DashboardResponse
		serviceErr       error
		wantStatus       int
		wantBodySubstr   string
		wantServiceCalls int
		wantLifecycle    int
	}{
		{
			name:             "GET returns 200 with JSON body and dispatches lifecycle",
			method:           http.MethodGet,
			id:               "NO",
			apiKey:           "key-123",
			serviceResp:      okResp,
			wantStatus:       http.StatusOK,
			wantBodySubstr:   `"isoCode":"NO"`,
			wantServiceCalls: 1,
			wantLifecycle:    1,
		},
		{name: "POST rejected with 405", method: http.MethodPost, id: "NO", wantStatus: http.StatusMethodNotAllowed},
		{name: "PUT rejected with 405", method: http.MethodPut, id: "NO", wantStatus: http.StatusMethodNotAllowed},
		{
			name:             "service error returns 400",
			method:           http.MethodGet,
			id:               "BAD",
			serviceErr:       errors.New("lookup failed"),
			wantStatus:       http.StatusBadRequest,
			wantBodySubstr:   "lookup failed",
			wantServiceCalls: 1,
		},
		{
			name:             "nil service response returns 404",
			method:           http.MethodGet,
			id:               "ZZ",
			serviceResp:      nil,
			wantStatus:       http.StatusNotFound,
			wantServiceCalls: 1,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeDashboardService{resp: tc.serviceResp, err: tc.serviceErr}

			req := httptest.NewRequest(tc.method, "/dashboards/"+tc.id, nil)
			req.SetPathValue("id", tc.id) // mimics ServeMux path parameter extraction
			if tc.apiKey != "" {
				req.Header.Set("x-api-key", tc.apiKey)
			}
			w := httptest.NewRecorder()

			dashboardHandler(svc, nil)(w, req)

			assert.Equal(t, tc.wantStatus, w.Code, "status code")
			if tc.wantBodySubstr != "" {
				assert.Contains(t, w.Body.String(), tc.wantBodySubstr)
			}
			assert.Equal(t, tc.wantServiceCalls, svc.calls, "service call count")

			if tc.wantStatus == http.StatusOK {
				assert.Equal(t, config.APPLICATION_JSON, w.Header().Get(config.HEADER_CONTENT_TYPE))
				assert.Equal(t, tc.id, svc.lastID, "path id reached service")
				assert.Equal(t, tc.apiKey, svc.lastAPIKey, "api key header reached service")

				var decoded structs.DashboardResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &decoded))
				assert.Equal(t, "NO", decoded.IsoCode)
			}
		})
	}
}
