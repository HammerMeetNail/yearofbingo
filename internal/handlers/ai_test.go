package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/HammerMeetNail/yearofbingo/internal/models"
	"github.com/HammerMeetNail/yearofbingo/internal/services/ai"
)

// MockAIService implements AIService interface
type MockAIService struct {
	GenerateGoalsFunc func(ctx context.Context, userID uuid.UUID, prompt ai.GoalPrompt) ([]string, ai.UsageStats, error)
	ConsumeFunc       func(ctx context.Context, userID uuid.UUID) (int, error)
}

func (m *MockAIService) GenerateGoals(ctx context.Context, userID uuid.UUID, prompt ai.GoalPrompt) ([]string, ai.UsageStats, error) {
	return m.GenerateGoalsFunc(ctx, userID, prompt)
}

func (m *MockAIService) ConsumeUnverifiedFreeGeneration(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.ConsumeFunc == nil {
		return 0, nil
	}
	return m.ConsumeFunc(ctx, userID)
}

func TestGenerate(t *testing.T) {
	// Setup common variables
	validBody := map[string]any{
		"category":   "hobbies",
		"focus":      "cooking",
		"difficulty": "medium",
		"budget":     "low",
		"context":    "none",
	}

	tests := []struct {
		name           string
		requestBody    interface{}
		user           *models.User
		mockSetup      func() *MockAIService
		expectedStatus int
	}{
		{
			name:        "Success",
			requestBody: validBody,
			user:        &models.User{ID: uuid.New(), EmailVerified: true},
			mockSetup: func() *MockAIService {
				return &MockAIService{
					GenerateGoalsFunc: func(ctx context.Context, userID uuid.UUID, prompt ai.GoalPrompt) ([]string, ai.UsageStats, error) {
						return []string{"Goal 1", "Goal 2"}, ai.UsageStats{}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success (Count defaults to 24)",
			requestBody: validBody,
			user:        &models.User{ID: uuid.New(), EmailVerified: true},
			mockSetup: func() *MockAIService {
				return &MockAIService{
					GenerateGoalsFunc: func(ctx context.Context, userID uuid.UUID, prompt ai.GoalPrompt) ([]string, ai.UsageStats, error) {
						if prompt.Count != 24 {
							t.Fatalf("expected count 24, got %d", prompt.Count)
						}
						return []string{"Goal 1", "Goal 2"}, ai.UsageStats{}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Success (Count passed through)",
			requestBody: map[string]any{
				"category":   "hobbies",
				"focus":      "cooking",
				"difficulty": "medium",
				"budget":     "low",
				"context":    "none",
				"count":      5,
			},
			user: &models.User{ID: uuid.New(), EmailVerified: true},
			mockSetup: func() *MockAIService {
				return &MockAIService{
					GenerateGoalsFunc: func(ctx context.Context, userID uuid.UUID, prompt ai.GoalPrompt) ([]string, ai.UsageStats, error) {
						if prompt.Count != 5 {
							t.Fatalf("expected count 5, got %d", prompt.Count)
						}
						return []string{"Goal 1", "Goal 2"}, ai.UsageStats{}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid Input - Count",
			requestBody: map[string]any{
				"category":   "hobbies",
				"difficulty": "medium",
				"budget":     "low",
				"count":      25,
			},
			user: &models.User{ID: uuid.New(), EmailVerified: true},
			mockSetup: func() *MockAIService {
				return &MockAIService{
					GenerateGoalsFunc: func(ctx context.Context, userID uuid.UUID, prompt ai.GoalPrompt) ([]string, ai.UsageStats, error) {
						t.Fatal("GenerateGoals should not be called when count is invalid")
						return nil, ai.UsageStats{}, nil
					},
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Success (Unverified consumes quota)",
			requestBody: validBody,
			user:        &models.User{ID: uuid.New(), EmailVerified: false},
			mockSetup: func() *MockAIService {
				return &MockAIService{
					ConsumeFunc: func(ctx context.Context, userID uuid.UUID) (int, error) {
						return 4, nil
					},
					GenerateGoalsFunc: func(ctx context.Context, userID uuid.UUID, prompt ai.GoalPrompt) ([]string, ai.UsageStats, error) {
						return []string{"Goal 1", "Goal 2"}, ai.UsageStats{}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Unverified blocked when quota exhausted",
			requestBody: validBody,
			user:        &models.User{ID: uuid.New(), EmailVerified: false},
			mockSetup: func() *MockAIService {
				return &MockAIService{
					ConsumeFunc: func(ctx context.Context, userID uuid.UUID) (int, error) {
						return 0, ai.ErrEmailVerificationRequired
					},
					GenerateGoalsFunc: func(ctx context.Context, userID uuid.UUID, prompt ai.GoalPrompt) ([]string, ai.UsageStats, error) {
						t.Fatal("GenerateGoals should not be called when quota is exhausted")
						return nil, ai.UsageStats{}, nil
					},
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:        "Unauthorized",
			requestBody: validBody,
			user:        nil,
			mockSetup: func() *MockAIService {
				return &MockAIService{}
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid Input - Category",
			requestBody: map[string]any{
				"category":   "invalid",
				"difficulty": "medium",
				"budget":     "low",
			},
			user: &models.User{ID: uuid.New(), EmailVerified: true},
			mockSetup: func() *MockAIService {
				return &MockAIService{}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid Input - Missing Difficulty",
			requestBody: map[string]any{
				"category": "hobbies",
				"budget":   "low",
			},
			user: &models.User{ID: uuid.New(), EmailVerified: true},
			mockSetup: func() *MockAIService {
				return &MockAIService{}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid Input - Difficulty",
			requestBody: map[string]any{
				"category":   "hobbies",
				"difficulty": "expert",
				"budget":     "low",
			},
			user: &models.User{ID: uuid.New(), EmailVerified: true},
			mockSetup: func() *MockAIService {
				return &MockAIService{}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid Input - Budget",
			requestBody: map[string]any{
				"category":   "hobbies",
				"difficulty": "medium",
				"budget":     "ultra",
			},
			user: &models.User{ID: uuid.New(), EmailVerified: true},
			mockSetup: func() *MockAIService {
				return &MockAIService{}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid Input - Focus Too Long",
			requestBody: map[string]any{
				"category":   "hobbies",
				"focus":      strings.Repeat("a", 101),
				"difficulty": "medium",
				"budget":     "low",
			},
			user: &models.User{ID: uuid.New(), EmailVerified: true},
			mockSetup: func() *MockAIService {
				return &MockAIService{}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid Input - Context Too Long",
			requestBody: map[string]any{
				"category":   "hobbies",
				"context":    strings.Repeat("a", 501),
				"difficulty": "medium",
				"budget":     "low",
			},
			user: &models.User{ID: uuid.New(), EmailVerified: true},
			mockSetup: func() *MockAIService {
				return &MockAIService{}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Invalid Input - Unknown Field",
			requestBody: `{"category":"hobbies","difficulty":"medium","budget":"low","unknown":true}`,
			user:        &models.User{ID: uuid.New(), EmailVerified: true},
			mockSetup: func() *MockAIService {
				return &MockAIService{}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Service Error - Safety",
			requestBody: validBody,
			user:        &models.User{ID: uuid.New(), EmailVerified: true},
			mockSetup: func() *MockAIService {
				return &MockAIService{
					GenerateGoalsFunc: func(ctx context.Context, userID uuid.UUID, prompt ai.GoalPrompt) ([]string, ai.UsageStats, error) {
						return nil, ai.UsageStats{}, ai.ErrSafetyViolation
					},
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Service Error - Rate Limit",
			requestBody: validBody,
			user:        &models.User{ID: uuid.New(), EmailVerified: true},
			mockSetup: func() *MockAIService {
				return &MockAIService{
					GenerateGoalsFunc: func(ctx context.Context, userID uuid.UUID, prompt ai.GoalPrompt) ([]string, ai.UsageStats, error) {
						return nil, ai.UsageStats{}, ai.ErrRateLimitExceeded
					},
				}
			},
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name:        "Service Error - Unavailable",
			requestBody: validBody,
			user:        &models.User{ID: uuid.New(), EmailVerified: true},
			mockSetup: func() *MockAIService {
				return &MockAIService{
					GenerateGoalsFunc: func(ctx context.Context, userID uuid.UUID, prompt ai.GoalPrompt) ([]string, ai.UsageStats, error) {
						return nil, ai.UsageStats{}, ai.ErrAIProviderUnavailable
					},
				}
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:        "Service Error - Not Configured",
			requestBody: validBody,
			user:        &models.User{ID: uuid.New(), EmailVerified: true},
			mockSetup: func() *MockAIService {
				return &MockAIService{
					GenerateGoalsFunc: func(ctx context.Context, userID uuid.UUID, prompt ai.GoalPrompt) ([]string, ai.UsageStats, error) {
						return nil, ai.UsageStats{}, ai.ErrAINotConfigured
					},
				}
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:        "Service Error - Generic",
			requestBody: validBody,
			user:        &models.User{ID: uuid.New(), EmailVerified: true},
			mockSetup: func() *MockAIService {
				return &MockAIService{
					GenerateGoalsFunc: func(ctx context.Context, userID uuid.UUID, prompt ai.GoalPrompt) ([]string, ai.UsageStats, error) {
						return nil, ai.UsageStats{}, errors.New("unknown error")
					},
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "Unverified Usage Tracking Unavailable",
			requestBody: validBody,
			user:        &models.User{ID: uuid.New(), EmailVerified: false},
			mockSetup: func() *MockAIService {
				return &MockAIService{
					ConsumeFunc: func(ctx context.Context, userID uuid.UUID) (int, error) {
						return 0, ai.ErrAIUsageTrackingUnavailable
					},
					GenerateGoalsFunc: func(ctx context.Context, userID uuid.UUID, prompt ai.GoalPrompt) ([]string, ai.UsageStats, error) {
						t.Fatal("GenerateGoals should not be called when tracking is unavailable")
						return nil, ai.UsageStats{}, nil
					},
				}
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:        "Unverified Usage Tracking Generic Error",
			requestBody: validBody,
			user:        &models.User{ID: uuid.New(), EmailVerified: false},
			mockSetup: func() *MockAIService {
				return &MockAIService{
					ConsumeFunc: func(ctx context.Context, userID uuid.UUID) (int, error) {
						return 0, errors.New("boom")
					},
					GenerateGoalsFunc: func(ctx context.Context, userID uuid.UUID, prompt ai.GoalPrompt) ([]string, ai.UsageStats, error) {
						t.Fatal("GenerateGoals should not be called when tracking fails")
						return nil, ai.UsageStats{}, nil
					},
				}
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := tt.mockSetup()
			handler := NewAIHandler(mockService)

			var bodyBytes []byte
			switch v := tt.requestBody.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}
			req := httptest.NewRequest("POST", "/api/ai/generate", bytes.NewBuffer(bodyBytes))

			// Mock context with user if provided
			if tt.user != nil {
				ctx := SetUserInContext(req.Context(), tt.user)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.Generate(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
