package routes

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"hospital-middleware/internal/clients"
	"hospital-middleware/internal/handlers"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/services"
	"hospital-middleware/internal/utils"
)

type patientStoreFake struct {
	rows        []models.Patient
	searchErr   error
	upsertErr   error
	searchCalls int
	upserted    *models.Patient
}

func (f *patientStoreFake) Search(_ context.Context, _ string, _ models.PatientFilters) ([]models.Patient, error) {
	f.searchCalls++
	return f.rows, f.searchErr
}

func (f *patientStoreFake) Upsert(_ context.Context, patient *models.Patient) error {
	f.upserted = patient
	return f.upsertErr
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPatientSearch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const hisPatient = `{"patient_hn":"A0001","national_id":"123","passport_id":"P123","first_name_th":"สมชาย","first_name_en":"Demo","date_of_birth":"1990-01-02"}`
	const responsePatient = `{"hn":"A0001","national_id":"123","passport_id":"P123","first_name_th":"สมชาย","first_name_en":"Demo","date_of_birth":"1990-01-02"}`
	nationalID, passportID := "123", "P123"
	birthTime, err := time.Parse(models.DateLayout, "1990-01-02")
	if err != nil {
		t.Fatal(err)
	}
	birthDate := models.Date(birthTime)
	cachedPatient := models.Patient{HN: "A0001", NationalID: &nationalID, PassportID: &passportID, FirstNameTH: "สมชาย", FirstNameEN: "Demo", DateOfBirth: &birthDate}

	tests := []struct {
		name           string
		hospital       string
		path           string
		cached         []models.Patient
		searchErr      error
		upsertErr      error
		upstreamStatus int
		upstreamError  error
		upstreamBody   string
		wantHISPath    string
		wantStatus     int
		wantBody       string
		wantHISCalls   int
		wantDBCalls    int
		wantUpsert     bool
	}{
		{name: "DB hit", hospital: "A", path: "/patient/search?national_id=123", cached: []models.Patient{cachedPatient}, wantStatus: 200, wantBody: `[` + responsePatient + `]`, wantDBCalls: 1},
		{name: "HIS hit", hospital: "A", path: "/patient/search?national_id=123", upstreamStatus: 200, wantHISPath: "/patient/search/123", wantStatus: 200, wantBody: `[` + responsePatient + `]`, wantDBCalls: 1, wantHISCalls: 1, wantUpsert: true},
		{name: "date filter matches HIS patient", hospital: "A", path: "/patient/search?national_id=123&date_of_birth=1990-01-02", upstreamStatus: 200, wantHISPath: "/patient/search/123", wantStatus: 200, wantBody: `[` + responsePatient + `]`, wantDBCalls: 1, wantHISCalls: 1, wantUpsert: true},
		{name: "date filter excludes HIS patient but still stores it", hospital: "A", path: "/patient/search?national_id=123&date_of_birth=1991-01-02", upstreamStatus: 200, wantHISPath: "/patient/search/123", wantStatus: 404, wantBody: `{"error":"patient not found"}`, wantDBCalls: 1, wantHISCalls: 1, wantUpsert: true},
		{name: "passport HIS hit", hospital: "A", path: "/patient/search?passport_id=P123", upstreamStatus: 200, wantHISPath: "/patient/search/P123", wantStatus: 200, wantBody: `[` + responsePatient + `]`, wantDBCalls: 1, wantHISCalls: 1, wantUpsert: true},
		{name: "no ID searches DB", hospital: "A", path: "/patient/search?first_name=Demo", cached: []models.Patient{cachedPatient}, wantStatus: 200, wantBody: `[` + responsePatient + `]`, wantDBCalls: 1},
		{name: "Thai name searches DB", hospital: "A", path: "/patient/search?first_name=%E0%B8%AA%E0%B8%A1%E0%B8%8A%E0%B8%B2%E0%B8%A2", cached: []models.Patient{cachedPatient}, wantStatus: 200, wantBody: `[` + responsePatient + `]`, wantDBCalls: 1},
		{name: "no ID and DB miss returns an empty list", hospital: "A", path: "/patient/search?first_name=Nobody", wantStatus: 200, wantBody: `[]`, wantDBCalls: 1},
		{name: "no filters at all returns an empty list", hospital: "A", path: "/patient/search", wantStatus: 200, wantBody: `[]`, wantDBCalls: 1},
		{name: "HIS not found", hospital: "A", path: "/patient/search?national_id=123", upstreamStatus: 404, wantHISPath: "/patient/search/123", wantStatus: 404, wantBody: `{"error":"patient not found"}`, wantDBCalls: 1, wantHISCalls: 1},
		{name: "unsupported hospital", hospital: "B", path: "/patient/search?national_id=123&hospital=A", wantStatus: 400, wantBody: `{"error":"unsupported hospital \"B\": only \"A\" has a connected HIS"}`},
		{name: "upstream failure", hospital: "A", path: "/patient/search?national_id=123", upstreamStatus: 503, wantHISPath: "/patient/search/123", wantStatus: 502, wantBody: `{"error":"Hospital HIS unavailable"}`, wantDBCalls: 1, wantHISCalls: 1},
		{name: "network failure", hospital: "A", path: "/patient/search?national_id=123", upstreamError: errors.New("connection failed"), wantHISPath: "/patient/search/123", wantStatus: 502, wantBody: `{"error":"Hospital HIS unavailable"}`, wantDBCalls: 1, wantHISCalls: 1},
		{name: "HIS returns a different patient", hospital: "A", path: "/patient/search?national_id=123", upstreamStatus: 200, upstreamBody: `{"patient_hn":"A0001","national_id":"999"}`, wantHISPath: "/patient/search/123", wantStatus: 502, wantBody: `{"error":"Hospital HIS unavailable"}`, wantDBCalls: 1, wantHISCalls: 1},
		{name: "HIS omits the ID we asked for", hospital: "A", path: "/patient/search?passport_id=P123", upstreamStatus: 200, upstreamBody: `{"patient_hn":"A0001","passport_id":"  "}`, wantHISPath: "/patient/search/P123", wantStatus: 502, wantBody: `{"error":"Hospital HIS unavailable"}`, wantDBCalls: 1, wantHISCalls: 1},
		{name: "invalid HIS date", hospital: "A", path: "/patient/search?national_id=123", upstreamStatus: 200, upstreamBody: `{"patient_hn":"A0001","national_id":"123","date_of_birth":"bad"}`, wantHISPath: "/patient/search/123", wantStatus: 502, wantBody: `{"error":"Hospital HIS unavailable"}`, wantDBCalls: 1, wantHISCalls: 1},
		{name: "invalid date filter", hospital: "A", path: "/patient/search?date_of_birth=bad", wantStatus: 400, wantBody: `{"error":"invalid search filters"}`},
		{name: "DB failure", hospital: "A", path: "/patient/search?national_id=123", searchErr: errors.New("db error"), wantStatus: 500, wantBody: `{"error":"patient storage failed"}`, wantDBCalls: 1},
		{name: "upsert failure", hospital: "A", path: "/patient/search?national_id=123", upstreamStatus: 200, wantHISPath: "/patient/search/123", upsertErr: errors.New("db error"), wantStatus: 500, wantBody: `{"error":"patient storage failed"}`, wantDBCalls: 1, wantHISCalls: 1, wantUpsert: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &patientStoreFake{rows: tt.cached, searchErr: tt.searchErr, upsertErr: tt.upsertErr}
			hisCalls := 0
			transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
				hisCalls++
				if r.Method != http.MethodGet || r.URL.Path != tt.wantHISPath {
					t.Errorf("unexpected HIS request: %s %s", r.Method, r.URL.Path)
				}
				if tt.upstreamError != nil {
					return nil, tt.upstreamError
				}
				body := ""
				if tt.upstreamStatus == 200 {
					body = hisPatient
				}
				if tt.upstreamBody != "" {
					body = tt.upstreamBody
				}
				return &http.Response{StatusCode: tt.upstreamStatus, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			jwtUtil := &utils.JwtUtil{SecretKey: []byte("test-secret"), Expiration: time.Hour}
			token, err := jwtUtil.GenerateToken(1, tt.hospital)
			if err != nil {
				t.Fatal(err)
			}
			patientHandler := &handlers.PatientHandler{Service: &services.PatientService{
				Repo:      store,
				HospitalA: &clients.HospitalAClient{BaseURL: "https://hospital-a.test", HTTPClient: &http.Client{Transport: transport}},
			}}
			router := gin.New()
			SetupRoutes(router, &handlers.StaffHandler{}, patientHandler, jwtUtil)
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != tt.wantStatus || strings.TrimSpace(response.Body.String()) != tt.wantBody {
				t.Fatalf("expected %d %s; got %d %s", tt.wantStatus, tt.wantBody, response.Code, response.Body.String())
			}
			if hisCalls != tt.wantHISCalls || store.searchCalls != tt.wantDBCalls || (store.upserted != nil) != tt.wantUpsert {
				t.Fatalf("unexpected calls: HIS=%d DB=%d upsert=%t", hisCalls, store.searchCalls, store.upserted != nil)
			}
			if store.upserted != nil && store.upserted.Hospital != "A" {
				t.Fatalf("wrong patient hospital: %q", store.upserted.Hospital)
			}
		})
	}
}
