package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/HammerMeetNail/yearofbingo/internal/config"
	"github.com/HammerMeetNail/yearofbingo/internal/handlers"
)

func TestRegisterAIRoutes_GlobalSwitch(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(strconv.FormatBool(enabled), func(t *testing.T) {
			mux := http.NewServeMux()
			sessions := 0
			requireSession := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					sessions++
					next.ServeHTTP(w, r)
				})
			}
			// Disabled routes must work without any service or rate limiter. If either
			// middleware or handlers are reached, the nil dependencies fail the test.
			if enabled {
				registerAIRoutes(mux, handlers.NewAIHandler(nil), true, requireSession, testRateLimiter(), testRateLimiter())
			} else {
				registerAIRoutes(mux, nil, false, requireSession, nil, nil)
			}
			mux.HandleFunc("GET /unrelated", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
			for _, path := range []string{"generate", "guide", "premium/status", "assist", "regenerate", "fill-empty"} {
				method := http.MethodPost
				if path == "premium/status" {
					method = http.MethodGet
				}
				rr := httptest.NewRecorder()
				mux.ServeHTTP(rr, httptest.NewRequest(method, "/api/ai/"+path, nil))
				want := http.StatusServiceUnavailable
				if enabled {
					want = http.StatusBadRequest
					if path == "premium/status" {
						want = http.StatusUnauthorized
					}
				}
				if rr.Code != want {
					t.Fatalf("%s status=%d want=%d: %s", path, rr.Code, want, rr.Body.String())
				}
				if !enabled {
					var body map[string]string
					if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
						t.Fatal(err)
					}
					if body["error"] != "AI features are currently disabled" {
						t.Fatalf("unexpected response: %v", body)
					}
				}
			}
			wantSessions := 0
			if enabled {
				wantSessions = 6
			}
			if sessions != wantSessions {
				t.Fatalf("session calls=%d want=%d", sessions, wantSessions)
			}
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/unrelated", nil))
			if rr.Code != http.StatusNoContent {
				t.Fatalf("unrelated route status=%d", rr.Code)
			}
		})
	}
}

func TestConfiguredFeatureSwitches(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, enhancements := range []bool{false, true} {
			cfg := &config.Config{
				AI:      config.AIConfig{Enabled: enabled},
				Billing: config.BillingConfig{FeatureTemplatesEnabled: true, FeatureEditAfterFinalizeEnabled: true, FeatureAIEnhancementsEnabled: enhancements},
			}
			got := configuredFeatureSwitches(cfg)
			if got.AIEnhancements != (enabled && enhancements) {
				t.Fatalf("AI enabled=%t enhancements=%t: got %+v", enabled, enhancements, got)
			}
			if !got.Templates || !got.EditAfterFinalize {
				t.Fatalf("non-AI premium features affected: %+v", got)
			}
		}
	}
}
