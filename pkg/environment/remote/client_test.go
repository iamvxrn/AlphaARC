package remote

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"protaxon/pkg/environment"
)

func TestNewClientFromEnvRequiresAPIKey(t *testing.T) {
	t.Setenv("ARC_API_KEY", "")
	t.Setenv("ARC_BASE_URL", "")
	if _, err := NewClientFromEnv(); err == nil {
		t.Fatalf("FAIL: expected an error when ARC_API_KEY is unset, got nil")
	}
}

func TestNewClientFromEnvUsesDefaultBaseURL(t *testing.T) {
	t.Setenv("ARC_API_KEY", "test-key")
	t.Setenv("ARC_BASE_URL", "")
	c, err := NewClientFromEnv()
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	if c.baseURL != "https://arcprize.org" {
		t.Fatalf("FAIL: expected default base URL https://arcprize.org, got %q", c.baseURL)
	}
	if c.apiKey != "test-key" {
		t.Fatalf("FAIL: expected apiKey to be read from ARC_API_KEY")
	}
}

func TestNewClientFromEnvHonorsOverrideBaseURL(t *testing.T) {
	t.Setenv("ARC_API_KEY", "test-key")
	t.Setenv("ARC_BASE_URL", "http://localhost:8001/")
	c, err := NewClientFromEnv()
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	if c.baseURL != "http://localhost:8001" {
		t.Fatalf("FAIL: expected trailing slash trimmed, got %q", c.baseURL)
	}
}

func TestListGames(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/games" {
			t.Fatalf("FAIL: unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "secret-key" {
			t.Fatalf("FAIL: expected X-API-Key header 'secret-key', got %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("FAIL: expected Accept: application/json, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{
			{"game_id": "ls20-016295f7f256"},
			{"game_id": "vc33-abc123"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret-key", nil)
	games, err := c.ListGames(context.Background())
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	if len(games) != 2 {
		t.Fatalf("FAIL: expected 2 games, got %d", len(games))
	}
	if games[0].GameID != "ls20-016295f7f256" || games[1].GameID != "vc33-abc123" {
		t.Fatalf("FAIL: unexpected game IDs: %+v", games)
	}
}

func TestOpenScorecard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/scorecard/open" {
			t.Fatalf("FAIL: unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("FAIL: could not decode request body: %v", err)
		}
		tags, ok := body["tags"].([]any)
		if !ok || len(tags) != 1 || tags[0] != "protaxon" {
			t.Fatalf("FAIL: expected tags [protaxon], got %+v", body)
		}
		json.NewEncoder(w).Encode(openScorecardResponse{CardID: "card-123"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret-key", nil)
	cardID, err := c.OpenScorecard(context.Background(), []string{"protaxon"})
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	if cardID != "card-123" {
		t.Fatalf("FAIL: expected card-123, got %q", cardID)
	}
}

func TestCloseScorecard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/scorecard/close" {
			t.Fatalf("FAIL: unexpected path %s", r.URL.Path)
		}
		var body closeScorecardRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.CardID != "card-123" {
			t.Fatalf("FAIL: expected card_id card-123 in request, got %q", body.CardID)
		}
		json.NewEncoder(w).Encode(map[string]any{"card_id": "card-123", "won": 1, "played": 3})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret-key", nil)
	summary, err := c.CloseScorecard(context.Background(), "card-123")
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	if summary["card_id"] != "card-123" {
		t.Fatalf("FAIL: expected card_id in response summary, got %+v", summary)
	}
}

// mockFrameResponse is the exact JSON shape confirmed from api.py's cmd()
// handler: game_id, levels_completed, win_levels, frame (a LIST of grids),
// state (enum .name), guid, action_input, available_actions.
func mockFrameResponse(state, guid string) map[string]any {
	return map[string]any{
		"game_id":           "ls20-test",
		"levels_completed":  0,
		"win_levels":        0,
		"frame":             [][][]int{{{0, 1}, {2, 3}}},
		"state":             state,
		"guid":              guid,
		"action_input":      map[string]any{"id": 0},
		"available_actions": []int{1, 2, 3, 4},
	}
}

func TestSessionResetSendsGameIDAndCardIDNoGUID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cmd/RESET" {
			t.Fatalf("FAIL: expected /api/cmd/RESET, got %s", r.URL.Path)
		}
		// Decode into a raw map, not actionInput -- decoding with the same
		// struct the client used to encode would stay symmetric even if a
		// json tag were typo'd, so this checks the literal wire keys.
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["game_id"] != "ls20-test" || body["card_id"] != "card-123" {
			t.Fatalf("FAIL: unexpected reset body: %+v", body)
		}
		if _, present := body["guid"]; present {
			t.Fatalf("FAIL: expected no guid key on first RESET (omitempty), got %+v", body)
		}
		json.NewEncoder(w).Encode(mockFrameResponse("NOT_FINISHED", "session-guid-1"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret-key", nil)
	sess := NewSession(c, "ls20-test", "card-123")

	frame, err := sess.Reset()
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	if frame.State != environment.StateNotFinished {
		t.Fatalf("FAIL: expected NOT_FINISHED, got %s", frame.State)
	}
	if len(frame.Grid) != 2 || len(frame.Grid[0]) != 2 {
		t.Fatalf("FAIL: expected 2x2 grid from first frame entry, got %+v", frame.Grid)
	}
	if sess.guid != "session-guid-1" {
		t.Fatalf("FAIL: expected session to capture guid from response, got %q", sess.guid)
	}
}

func TestSessionStepRequiresResetFirst(t *testing.T) {
	c := NewClient("http://unused.invalid", "secret-key", nil)
	sess := NewSession(c, "ls20-test", "card-123")

	_, err := sess.Step(environment.Action{ID: environment.Action1})
	if err == nil {
		t.Fatalf("FAIL: expected error calling Step before Reset")
	}
}

func TestSessionStepSendsGUIDAndUpdatesItFromResponse(t *testing.T) {
	var callCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Path != "/api/cmd/ACTION1" {
			t.Fatalf("FAIL: expected /api/cmd/ACTION1, got %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["guid"] != "session-guid-1" {
			t.Fatalf("FAIL: expected guid session-guid-1 to be forwarded, got %+v", body)
		}
		json.NewEncoder(w).Encode(mockFrameResponse("NOT_FINISHED", "session-guid-2"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret-key", nil)
	sess := NewSession(c, "ls20-test", "card-123")
	sess.guid = "session-guid-1" // simulate a completed Reset

	_, err := sess.Step(environment.Action{ID: environment.Action1})
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("FAIL: expected exactly 1 request, got %d", callCount)
	}
	if sess.guid != "session-guid-2" {
		t.Fatalf("FAIL: expected session guid to advance to session-guid-2, got %q", sess.guid)
	}
}

func TestSessionStepSendsXYOnlyForComplexAction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cmd/ACTION6" {
			t.Fatalf("FAIL: expected /api/cmd/ACTION6, got %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		x, xOK := body["x"].(float64)
		y, yOK := body["y"].(float64)
		if !xOK || !yOK {
			t.Fatalf("FAIL: expected x/y keys to be present for ACTION6, got %+v", body)
		}
		if x != 12 || y != 34 {
			t.Fatalf("FAIL: expected x=12 y=34, got x=%v y=%v", x, y)
		}
		json.NewEncoder(w).Encode(mockFrameResponse("NOT_FINISHED", "session-guid-2"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret-key", nil)
	sess := NewSession(c, "ls20-test", "card-123")
	sess.guid = "session-guid-1"

	_, err := sess.Step(environment.Action{ID: environment.Action6, X: 12, Y: 34})
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
}

// TestSessionStepOmitsXYForSimpleAction closes the coverage gap the
// previous test's name implied but didn't check: a non-complex action
// (e.g. ACTION1) must NOT send x/y keys at all, not even as null/zero.
func TestSessionStepOmitsXYForSimpleAction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if _, present := body["x"]; present {
			t.Fatalf("FAIL: expected no x key for a simple action, got %+v", body)
		}
		if _, present := body["y"]; present {
			t.Fatalf("FAIL: expected no y key for a simple action, got %+v", body)
		}
		json.NewEncoder(w).Encode(mockFrameResponse("NOT_FINISHED", "session-guid-2"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret-key", nil)
	sess := NewSession(c, "ls20-test", "card-123")
	sess.guid = "session-guid-1"

	_, err := sess.Step(environment.Action{ID: environment.Action1})
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
}

func TestSessionStepUnknownActionID(t *testing.T) {
	c := NewClient("http://unused.invalid", "secret-key", nil)
	sess := NewSession(c, "ls20-test", "card-123")
	sess.guid = "session-guid-1"

	_, err := sess.Step(environment.Action{ID: environment.ActionID(99)})
	if err == nil {
		t.Fatalf("FAIL: expected error for unknown action ID")
	}
}

func TestNonSuccessStatusReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized","message":"invalid X-API-Key"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "bad-key", nil)
	_, err := c.ListGames(context.Background())
	if err == nil {
		t.Fatalf("FAIL: expected an error on HTTP 401")
	}
}

func TestSessionWinState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mockFrameResponse("WIN", "session-guid-2"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret-key", nil)
	sess := NewSession(c, "ls20-test", "card-123")
	sess.guid = "session-guid-1"

	frame, err := sess.Step(environment.Action{ID: environment.Action1})
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	if frame.State != environment.StateWin {
		t.Fatalf("FAIL: expected StateWin, got %s", frame.State)
	}
}
