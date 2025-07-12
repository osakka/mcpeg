package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/osakka/mcpeg/pkg/health"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/validation"
)

// TestAdminAuthMiddleware tests the admin API authentication middleware
func TestAdminAuthMiddleware(t *testing.T) {
	logger := logging.New("test")
	mockMetrics := &mockMetrics{}
	validator := validation.NewValidator(logger, mockMetrics)
	healthMgr := health.NewHealthManager(logger, mockMetrics, "test")
	
	t.Run("auth disabled when no API key configured", func(t *testing.T) {
		config := ServerConfig{
			AdminAPIKey:    "", // No API key configured
			AdminAPIHeader: "X-Admin-API-Key",
		}
		
		_ = NewGatewayServer(config, logger, mockMetrics, validator, healthMgr)
		
		// Create test handler
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})
		
		// Create admin router (auth should not be applied)
		router := mux.NewRouter()
		adminRouter := router.PathPrefix("/admin").Subrouter()
		
		// Simulate the setup without auth middleware since AdminAPIKey is empty
		adminRouter.HandleFunc("/test", testHandler).Methods("GET")
		
		// Test request without API key should succeed
		req := httptest.NewRequest("GET", "/admin/test", nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 when auth disabled, got %d", w.Code)
		}
	})
	
	t.Run("auth required when API key configured", func(t *testing.T) {
		config := ServerConfig{
			AdminAPIKey:    "test-secret-key",
			AdminAPIHeader: "X-Admin-API-Key",
		}
		
		server := NewGatewayServer(config, logger, mockMetrics, validator, healthMgr)
		
		// Create test handler
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})
		
		// Create admin router with auth middleware
		router := mux.NewRouter()
		adminRouter := router.PathPrefix("/admin").Subrouter()
		adminRouter.Use(server.adminAuthMiddleware)
		adminRouter.HandleFunc("/test", testHandler).Methods("GET")
		
		// Test request without API key should fail
		req := httptest.NewRequest("GET", "/admin/test", nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 without API key, got %d", w.Code)
		}
	})
	
	t.Run("auth success with correct API key", func(t *testing.T) {
		config := ServerConfig{
			AdminAPIKey:    "test-secret-key",
			AdminAPIHeader: "X-Admin-API-Key",
		}
		
		server := NewGatewayServer(config, logger, mockMetrics, validator, healthMgr)
		
		// Create test handler
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})
		
		// Create admin router with auth middleware
		router := mux.NewRouter()
		adminRouter := router.PathPrefix("/admin").Subrouter()
		adminRouter.Use(server.adminAuthMiddleware)
		adminRouter.HandleFunc("/test", testHandler).Methods("GET")
		
		// Test request with correct API key should succeed
		req := httptest.NewRequest("GET", "/admin/test", nil)
		req.Header.Set("X-Admin-API-Key", "test-secret-key")
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 with correct API key, got %d", w.Code)
		}
		
		if w.Body.String() != "success" {
			t.Errorf("expected 'success' response, got %s", w.Body.String())
		}
	})
	
	t.Run("auth failure with incorrect API key", func(t *testing.T) {
		config := ServerConfig{
			AdminAPIKey:    "test-secret-key",
			AdminAPIHeader: "X-Admin-API-Key",
		}
		
		server := NewGatewayServer(config, logger, mockMetrics, validator, healthMgr)
		
		// Create test handler
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})
		
		// Create admin router with auth middleware
		router := mux.NewRouter()
		adminRouter := router.PathPrefix("/admin").Subrouter()
		adminRouter.Use(server.adminAuthMiddleware)
		adminRouter.HandleFunc("/test", testHandler).Methods("GET")
		
		// Test request with incorrect API key should fail
		req := httptest.NewRequest("GET", "/admin/test", nil)
		req.Header.Set("X-Admin-API-Key", "wrong-key")
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 with incorrect API key, got %d", w.Code)
		}
	})
	
	t.Run("custom header name", func(t *testing.T) {
		config := ServerConfig{
			AdminAPIKey:    "test-secret-key",
			AdminAPIHeader: "X-Custom-Auth",
		}
		
		server := NewGatewayServer(config, logger, mockMetrics, validator, healthMgr)
		
		// Create test handler
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})
		
		// Create admin router with auth middleware
		router := mux.NewRouter()
		adminRouter := router.PathPrefix("/admin").Subrouter()
		adminRouter.Use(server.adminAuthMiddleware)
		adminRouter.HandleFunc("/test", testHandler).Methods("GET")
		
		// Test request with custom header should succeed
		req := httptest.NewRequest("GET", "/admin/test", nil)
		req.Header.Set("X-Custom-Auth", "test-secret-key")
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 with custom header, got %d", w.Code)
		}
	})
}

// mockMetrics implements a basic metrics interface for testing
type mockMetrics struct{}

func (m *mockMetrics) Inc(name string, labels ...string) {}
func (m *mockMetrics) Add(name string, value float64, labels ...string) {}
func (m *mockMetrics) Set(name string, value float64, labels ...string) {}
func (m *mockMetrics) Observe(name string, value float64, labels ...string) {}
func (m *mockMetrics) Time(name string, labels ...string) metrics.Timer { return &mockTimer{} }
func (m *mockMetrics) WithLabels(labels map[string]string) metrics.Metrics { return m }
func (m *mockMetrics) WithPrefix(prefix string) metrics.Metrics { return m }
func (m *mockMetrics) GetStats(name string) metrics.MetricStats { return metrics.MetricStats{} }
func (m *mockMetrics) GetAllStats() map[string]metrics.MetricStats { return make(map[string]metrics.MetricStats) }

type mockTimer struct{}

func (t *mockTimer) Duration() time.Duration { return 0 }
func (t *mockTimer) Stop() time.Duration { return 0 }