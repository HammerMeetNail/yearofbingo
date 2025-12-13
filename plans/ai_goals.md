# AI Goal Wizard Implementation Plan

## Overview

**Goal:** Implement an interactive "Wizard" workflow that generates personalized Bingo cards using Generative AI.
**Core Value:** Solves the "blank canvas" problem by converting user demographics and interests into specific, achievable, and safe goals.

## User Stories

1.  **As a new user**, I want to describe my interests (e.g., "cooking," "hiking") so that I don't have to think of 24 unique goals from scratch.
2.  **As a user**, I want to choose the "difficulty" and "time commitment" of my card so that the goals fit my lifestyle.
3.  **As a user**, I want to review the AI-generated goals and swap out ones I don't like before creating the card.
4.  **As a system owner**, I want to ensure generated content is safe (no NSFW/hate speech) and that the system cannot be abused to bypass LLM costs.

## Design Decisions & Evaluation

### 1. AI Provider Selection

We evaluated three primary contenders for cost-effective, high-speed generation:

| Provider | Model | Cost (Input/Output per 1M tokens) | Speed | Safety Filters | Verdict |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Google** | **Gemini 2.5 Flash-Lite** | **$0.10 / $0.40** (Paid) or **FREE** (Tier) | **Fastest** | **Excellent (Native)** | **WINNER** |
| Google | Gemini 2.5 Flash | $0.30 / $2.50 | Fast | Excellent | Too expensive |
| OpenAI | GPT-4o-mini | $0.15 / $0.60 | Fast | Good | Runner Up |

**Why Gemini 2.5 Flash-Lite?**
-   **Cost:** Lowest cost per request ($0.10/$0.40 vs $0.30/$2.50 for standard Flash).
-   **Structured Output:** Excellent support for JSON schema enforcement.
-   **Safety:** Built-in configurable safety settings.

### 2. Architecture: Hosted Proxy vs. BYOK (Bring Your Own Key)

**Decision: Hosted Proxy (Backend-driven)**

*   **User Experience:** "Bring Your Own Key" adds massive friction. Non-technical users do not have API keys. A wizard should be magical, not administrative.
*   **Security:** API keys must never be exposed to the client. A backend proxy is required regardless.
*   **Safety:** We must enforce the System Prompt to ensure safety. Allowing users to supply their own key might imply they can supply their own prompts, which breaks the "Bingo" constraint.
*   **Cost Control:** With Gemini Flash, the cost is so low that subsidizing it is feasible. We will mitigate abuse via Rate Limiting.

### 3. Data Persistence

*   **User Input Data:** **Transient**. We do *not* store the specific demographic answers (Age, Gender, Specific Interests) in the database to minimize privacy risks. This data exists only in memory during the request lifecycle.
*   **Generated Goals:** Stored only when the user clicks "Create Card" (saved as standard `CardItem` rows).
*   **Wizard State:** Persisted in `localStorage` on the client side to allow "resuming" if the user accidentally navigates away.

## Gemini API Integration Details

### Endpoint
`POST https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-lite:generateContent?key=${GEMINI_API_KEY}`

### Request Payload
We use `generationConfig` to enforce a JSON array of strings as the output.

```json
{
  "contents": [
    {
      "parts": [
        {
          "text": "SYSTEM_PROMPT\n\nUSER_CONTEXT_JSON"
        }
      ]
    }
  ],
  "generationConfig": {
    "responseMimeType": "application/json",
    "responseSchema": {
      "type": "array",
      "items": {
        "type": "string"
      }
    },
    "temperature": 1.0
  },
  "safetySettings": [
    { "category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_MEDIUM_AND_ABOVE" },
    { "category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_MEDIUM_AND_ABOVE" },
    { "category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_MEDIUM_AND_ABOVE" },
    { "category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_MEDIUM_AND_ABOVE" }
  ]
}
```

### Response Format

```json
{
  "candidates": [
    {
      "content": {
        "parts": [
          {
            "text": "[\"Goal 1\", \"Goal 2\", ...]"
          }
        ]
      },
      "finishReason": "STOP",
      "safetyRatings": [...]
    }
  ]
}
```

## Authentication & Data Privacy

### Authentication Strategy
We will use the **Gemini API Key** (via Google AI Studio) for authentication.
*   **Mechanism**: The key is passed as a query parameter `?key=${GEMINI_API_KEY}` in the HTTP request.
*   **Security**: The key is stored as a backend environment variable (`GEMINI_API_KEY`) and is **never** exposed to the client.

### Data Privacy & Training
*   **Terms of Service**: We will use a **Paid Tier** (Blaze plan) or an account with **Cloud Billing enabled**. Under these terms, Google **does not** use input data (prompts) or output data (responses) to train their models.
*   **Vertex AI Comparison**: While Vertex AI offers "Zero Data Retention" guarantees, the complexity of setting up GCP IAM, Service Accounts, and OAuth flows is excessive for this feature's current scope. The standard Gemini API's privacy terms for paid accounts are sufficient for consumer-grade goals (non-PII, non-HIPAA).
*   **Long-Term**: If the app pivots to enterprise or healthcare use cases, we will migrate to Vertex AI to leverage its enhanced compliance controls.

## Database Schema

### New Table: `ai_generation_logs` (Audit & Analytics)

While we don't store user PII, we need to track usage for rate limiting and cost monitoring.

```sql
CREATE TABLE ai_generation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL, -- Nullable for anonymous try-outs (if we allow that)
    model VARCHAR(50) NOT NULL,                           -- e.g. "gemini-1.5-flash"
    tokens_input INT NOT NULL,
    tokens_output INT NOT NULL,
    duration_ms INT NOT NULL,
    status VARCHAR(20) NOT NULL,                          -- 'success', 'error', 'safety_block'
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ai_logs_user_date ON ai_generation_logs(user_id, created_at);
```

## API Endpoints

### 1. Generate Goals

`POST /api/ai/generate`

**Auth:** Session required.
**Rate Limit:** 10 requests / hour / user.

**Request Payload:**
```json
{
  "category": "hobbies",       // "health", "career", "social", "mix"
  "focus": "cooking",          // Free text input
  "difficulty": "medium",      // "easy", "medium", "hard"
  "frequency": "weekly",       // "daily", "weekly", "monthly", "once"
  "context": "I want to learn italian cuisine" // Optional extra context
}
```

**Response:**
```json
{
  "goals": [
    "Cook a pasta dish from scratch",
    "Make homemade pizza dough",
    "Identify 3 different fresh herbs",
    ... (24 items)
  ],
  "usage_id": "uuid..." // Reference for feedback/logging
}
```

## Technical Architecture

### Backend (Go)

**Package:** `internal/services/ai`

**Interfaces:**
```go
type AIProvider interface {
    GenerateGoals(ctx context.Context, prompt GoalPrompt) ([]string, UsageStats, error)
}
```

**Implementation Details:**
-   **Prompt Engineering:**
    -   *System:* "You are an expert life coach creating a Bingo card. You must output exactly 24 distinct, short, achievable goals. Output JSON only."
    -   *Safety:* Explicit instruction to avoid dangerous, illegal, or sexually explicit content.
    -   *Format:* JSON Schema enforcement (Array of strings).
-   **Client:** standard `http.Client` calls to Google Generative Language API.

### Frontend (JS)

**State Management (`App.wizardState`):**
-   `step`: 'input' | 'loading' | 'review'
-   `inputs`: { ...user selections... }
-   `results`: [ ...strings... ]

## UI / UX Flow

### Step 1: "The Interview" (Input)
```
┌─────────────────────────────────────────────────────────────┐
│ 🧙 Bingo Wizard                                             │
├─────────────────────────────────────────────────────────────┤
│ What area of life is this for?                              │
│ [ ▼ Hobbies ]                                               │
│                                                             │
│ What specifically? (e.g. "Hiking", "Coding")                │
│ [_______________________]                                   │
│                                                             │
│ How intense?                                                │
│ ( ) Chill   (●) Balanced   ( ) Hardcore                     │
│                                                             │
│ Any other details?                                          │
│ [ I have a budget of $50/month...       ]                   │
│                                                             │
│                   [ Generate Goals ✨ ]                     │
└─────────────────────────────────────────────────────────────┘
```

### Step 2: "The Review" (Refinement)
```
┌─────────────────────────────────────────────────────────────┐
│ 🔮 Your Generated Card                                      │
├─────────────────────────────────────────────────────────────┤
│ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ │
│ │ Walk 5k │ │ New Rec │ │ ...     │ │ ...     │ │ ...     │ │
│ └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘ │
│ ... (Grid of 24 items) ...                                  │
│                                                             │
│ [ Regenerate All ↻ ]   [ Click item to edit ]               │
│                                                             │
│                   [ Create Card → ]                         │
└─────────────────────────────────────────────────────────────┘
```

## Implementation Phases

### Phase 0: Configuration & Secrets
1.  [ ] **Environment**: Update `.env.example` to include `GEMINI_API_KEY`.
2.  [ ] **Compose Files**: Update `compose.yaml` (local), `compose.prod.yaml`, and `compose.server.yaml` to inject `GEMINI_API_KEY`.
3.  [ ] **Config Loader**: Update `internal/config/config.go` to load the key into the `Config` struct.
4.  [ ] **CI/CD**: Update GitHub Secrets to include `GEMINI_API_KEY`. Update `.github/workflows/ci.yaml` to inject this secret into the `.env.production` file created during deployment.

### Phase 1: Infrastructure & Service
1.  [ ] **Migration**: Create `ai_generation_logs` table.
2.  [ ] **Service**: Implement `internal/services/ai/gemini.go`.
    -   Define prompt templates.
    -   Implement JSON parsing and error handling.

### Phase 2: API Layer
1.  [ ] **Handler**: Create `internal/handlers/ai.go`.
2.  [ ] **Rate Limiting**: Add Redis-based sliding window limiter (10 req/hr).
3.  [ ] **Route**: Register `POST /api/ai/generate`.
4.  [ ] **Tests**: Unit tests for prompt construction and mock API responses.
5.  [ ] **Integration Tests**: End-to-end API tests with VCR/cassette recording.

### Phase 3: Frontend Implementation
1.  [ ] **API Client**: Add `generateGoals` method to `web/static/js/api.js`.
2.  [ ] **Wizard Logic**: Implement `web/static/js/ai-wizard.js` (State management & UI rendering).
3.  [ ] **Integration**: Hook Wizard into the "Create Card" flow in `web/static/js/app.js`.
4.  [ ] **Tests**: Add frontend tests for the Wizard flow.

## Testing Strategy

Given the external dependency (AI API) and the non-deterministic nature of LLMs, testing requires a robust strategy that balances reliability with cost.

### 1. Unit Tests (`internal/services/ai/`)
*   **Prompt Construction**: Verify that user inputs (category, focus, difficulty) are correctly formatted into the final prompt string.
*   **Response Parsing**: Test the JSON parsing logic against various mock AI responses (valid JSON, malformed JSON, empty arrays).
*   **Error Handling**: Ensure the service correctly maps API errors (429, 500, Safety Violations) to internal application errors.

### 2. Integration Tests (`tests/integration/`)
*   **Mocked Provider**: The primary integration tests for the HTTP handler will use a `MockAIProvider` that returns pre-canned responses. This ensures the HTTP layer, Rate Limiter, and Logging middleware work without calling the real API.
*   **"Live" Tests (with Recording)**: We will create a test that *can* hit the real API but uses a VCR-like pattern (saving the response to a file) to ensure the contract with Google Gemini remains valid.
    *   *Note*: These will be run manually or on a schedule, not on every PR commit, to save costs and avoid flakiness.

### 3. Frontend Tests (`web/static/js/tests/`)
*   **State Management**: Unit tests for the wizard state reducer (handling user input updates).
*   **Component Rendering**: Verify the "Wizard" form renders correct options.
*   **API Integration**: Mock `fetch` to verify `API.ai.generate` constructs the correct request body and handles the loading state transition.

### 4. QA & Safety Verification
*   **Safety Probe**: A specific manual test set where we attempt to feed "unsafe" inputs to the local dev environment to verify the safety filters trigger the correct error message.

## Error Handling Specification

We will strictly adhere to the project's existing error handling patterns (Service-defined errors -> Handler mapping).

### 1. Service Layer Errors (`internal/services/ai/errors.go`)
We will define specific sentinel errors to handle the nuances of AI interaction:

```go
var (
    ErrAIProviderUnavailable = errors.New("AI provider is currently unavailable") // 503
    ErrSafetyViolation       = errors.New("generated content violated safety policies") // 400
    ErrRateLimitExceeded     = errors.New("rate limit exceeded") // 429
    ErrInvalidInput          = errors.New("invalid input parameters") // 400
)
```

### 2. HTTP Handler Mapping (`internal/handlers/ai.go`)
The handler will catch these errors and map them to standard HTTP status codes using the existing `writeError` helper:

| Service Error | HTTP Status | User Message |
| :--- | :--- | :--- |
| `ErrAIProviderUnavailable` | `503 Service Unavailable` | "The AI service is currently down. Please try again later." |
| `ErrSafetyViolation` | `400 Bad Request` | "We couldn't generate safe goals for that topic. Please try rephrasing." |
| `ErrRateLimitExceeded` | `429 Too Many Requests` | "You've reached the limit for AI generations. Please try again in an hour." |
| `ErrInvalidInput` | `400 Bad Request` | "Invalid input provided." |
| *(Unknown Error)* | `500 Internal Server Error` | "An unexpected error occurred." |

### 3. JSON Response Format
All errors will return the standard JSON structure:
```json
{
  "error": "We couldn't generate safe goals for that topic. Please try rephrasing."
}
```

## Security & Safety Considerations

1.  **Input Injection**:
    -   User free-text input must be sanitized before insertion into the prompt to prevent "Ignore previous instructions" attacks.
    -   *Mitigation*: We will use the Gemini API's "structure" fields (System Instructions vs User Content) effectively to segregate instructions from data.
2.  **Output Safety**:
    -   We rely on Gemini's "Hate speech", "Harassment", and "Dangerous content" filters set to `BLOCK_MEDIUM_AND_ABOVE`.
    -   If the API returns a safety violation, we show a generic error: "We couldn't generate safe goals for that topic. Please try rephrasing."
3.  **Cost Denial of Service**:
    -   Strict Rate Limits (10/hr).
    -   Max token limits on the API request (prevent AI from generating novels).

## Cost Analysis

**Scenario: 1,000 Cards Generated per Month**

*   **Prompt Size**: ~300 tokens (System + User context).
*   **Response Size**: ~400 tokens (24 short sentences).
*   **Total Tokens**: 700 * 1,000 = 700,000 tokens.
*   **Gemini Flash Price**: ~$0.30 / 1M tokens (blended).
*   **Total Cost**: **$0.21 / month**.

**Conclusion**: The feature is effectively free to operate at our current scale.