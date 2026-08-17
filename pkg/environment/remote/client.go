// Package remote is a real HTTP client for the ARC-AGI-3 REST API.
//
// Every endpoint path, header, and request/response field named here was
// confirmed 2026-08-12 directly against the official server implementation
// (github.com/arcprize/ARC-AGI: arc_agi/server.py's Flask route table and
// arc_agi/api.py's RestAPI handlers), plus the official Python agent
// starter kit (github.com/arcprize/ARC-AGI-3-Agents: main.py, .env.example)
// for header names and the production base URL -- not guessed.
package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"

	"alphaarc/pkg/environment"
)

// Client is a REST client for the ARC-AGI-3 API.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// BaseURL returns the host this client sends requests to -- useful for
// callers that want to log/print which service they actually hit, since a
// wrong-but-reachable host can fail in confusing, endpoint-specific ways
// (see NewClientFromEnv's doc comment for exactly this happening once).
func (c *Client) BaseURL() string {
	return c.baseURL
}

// NewClient builds a client against an explicit base URL and key. Mainly
// for tests, which point baseURL at an httptest.Server instead of the real
// service.
//
// When httpClient is nil, the constructed client carries a cookie jar --
// required for the real service, confirmed 2026-08-13 via
// docs.arcprize.org/rest_overview: "Games are stateful and require session
// affinity... The server sets cookies (especially AWSALB* cookies)... These
// cookies route requests to the correct backend instance maintaining your
// game state." Without a jar, Go's http.Client silently drops every
// Set-Cookie it receives, so RESET/ACTION calls after the first request in
// a session can land on a different backend instance than the one holding
// that game's state -- this is exactly what caused a real, reproducible
// "game <id> not found" on POST /api/cmd/RESET for a game_id GET /api/games
// had just returned.
func NewClient(baseURL, apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		jar, _ := cookiejar.New(nil) // New never actually errors, with or without an *Options
		httpClient = &http.Client{Timeout: 30 * time.Second, Jar: jar}
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    httpClient,
	}
}

// NewClientFromEnv reads ARC_API_KEY (required) and ARC_BASE_URL (optional,
// defaults to the real production service) from the environment, mirroring
// the official Python SDK's .env convention. The key is held only inside
// the returned *Client and never logged.
//
// The default base URL is https://three.arcprize.org, NOT https://arcprize.org
// -- confirmed 2026-08-13 the hard way: a live run against arcprize.org got
// real responses from GET /api/games and POST /api/scorecard/open (so it
// looked right), but POST /api/cmd/RESET 400'd with "game <id> not found"
// for every game_id that same /api/games call had just returned. The
// ARC-AGI-3-Agents starter kit's own .env.example defaults to
// HOST=arcprize.org, which is what this client's default originally copied
// -- but the arc_agi toolkit's own README config table (arcprize/ARC-AGI)
// documents arc_base_url's real default as "https://three.arcprize.org",
// and that's the one that actually owns game session state. arcprize.org
// apparently serves (at least) games listing and scorecard creation from
// shared infrastructure, which is why those two calls silently succeeded
// against the wrong host while RESET didn't.
func NewClientFromEnv() (*Client, error) {
	apiKey := os.Getenv("ARC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ARC_API_KEY is not set in the environment")
	}
	baseURL := os.Getenv("ARC_BASE_URL")
	if baseURL == "" {
		baseURL = "https://three.arcprize.org"
	}
	return NewClient(baseURL, apiKey, nil), nil
}

// GameInfo is one entry from GET /api/games. Only game_id is confirmed
// from the source read so far (main.py: `g["game_id"]`); the server may
// send more fields, which are simply ignored here.
type GameInfo struct {
	GameID string `json:"game_id"`
}

// ListGames calls GET /api/games.
func (c *Client) ListGames(ctx context.Context) ([]GameInfo, error) {
	var games []GameInfo
	if err := c.do(ctx, http.MethodGet, "/api/games", nil, &games); err != nil {
		return nil, err
	}
	return games, nil
}

type openScorecardRequest struct {
	Tags []string `json:"tags,omitempty"`
}

type openScorecardResponse struct {
	CardID string `json:"card_id"`
}

// OpenScorecard calls POST /api/scorecard/open, returning the new card_id.
func (c *Client) OpenScorecard(ctx context.Context, tags []string) (string, error) {
	var resp openScorecardResponse
	if err := c.do(ctx, http.MethodPost, "/api/scorecard/open", openScorecardRequest{Tags: tags}, &resp); err != nil {
		return "", err
	}
	return resp.CardID, nil
}

type closeScorecardRequest struct {
	CardID string `json:"card_id"`
}

// CloseScorecard calls POST /api/scorecard/close. The response is the
// server's EnvironmentScorecard summary; its full field shape wasn't read
// from source, so it's returned as a raw map rather than a guessed struct.
func (c *Client) CloseScorecard(ctx context.Context, cardID string) (map[string]any, error) {
	var resp map[string]any
	if err := c.do(ctx, http.MethodPost, "/api/scorecard/close", closeScorecardRequest{CardID: cardID}, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// actionInput is the request body for POST /api/cmd/<ACTION>, matching
// arc_agi/api.py's cmd() handler: game_id is always required, guid is
// required for everything except RESET, card_id and reasoning are
// optional, and X/Y are only sent for ACTION6 (the one "complex" action).
type actionInput struct {
	GameID    string `json:"game_id"`
	CardID    string `json:"card_id,omitempty"`
	GUID      string `json:"guid,omitempty"`
	Reasoning any    `json:"reasoning,omitempty"`
	X         *int   `json:"x,omitempty"`
	Y         *int   `json:"y,omitempty"`
}

// frameResponse mirrors the JSON fields api.py's cmd() handler builds (the
// `update` dict) that this client actually needs: game_id,
// levels_completed, win_levels, frame, state (the GameState enum's .name
// string), guid, available_actions. The response also includes
// action_input, which is silently ignored here since nothing needs it.
//
// Frame is confirmed to be a LIST of 2D grids, not a single grid --
// agent.py converts it as `frame=[arr.tolist() for arr in raw.frame]`.
// What multiple entries mean (history vs. layers) wasn't confirmed from
// what's been read, so this stays a 3D slice instead of assuming length 1;
// toFrame() below takes the first entry as "the" current grid, which is
// the only interpretation that's actually been verified.
type frameResponse struct {
	GameID           string    `json:"game_id"`
	LevelsCompleted  int       `json:"levels_completed"`
	WinLevels        int       `json:"win_levels"`
	Frame            [][][]int `json:"frame"`
	State            string    `json:"state"`
	GUID             string    `json:"guid"`
	AvailableActions []int     `json:"available_actions"`
}

func (f frameResponse) toFrame() environment.Frame {
	var grid [][]int
	if len(f.Frame) > 0 {
		grid = f.Frame[0]
	}
	actions := make([]environment.ActionID, len(f.AvailableActions))
	for i, a := range f.AvailableActions {
		actions[i] = environment.ActionID(a)
	}
	return environment.Frame{
		GameID:           f.GameID,
		Grid:             grid,
		State:            environment.GameState(f.State),
		LevelsCompleted:  f.LevelsCompleted,
		AvailableActions: actions,
	}
}

// actionNames maps environment.ActionID to the exact route segment
// confirmed in server.py's route table (/api/cmd/RESET, /api/cmd/ACTION1
// .. /api/cmd/ACTION7).
var actionNames = map[environment.ActionID]string{
	environment.ActionReset: "RESET",
	environment.Action1:     "ACTION1",
	environment.Action2:     "ACTION2",
	environment.Action3:     "ACTION3",
	environment.Action4:     "ACTION4",
	environment.Action5:     "ACTION5",
	environment.Action6:     "ACTION6",
	environment.Action7Undo: "ACTION7",
}

// cmd calls POST /api/cmd/<name> and returns the decoded frame plus the
// session guid the server assigned/echoed back.
func (c *Client) cmd(ctx context.Context, name string, in actionInput) (environment.Frame, string, error) {
	var resp frameResponse
	if err := c.do(ctx, http.MethodPost, "/api/cmd/"+name, in, &resp); err != nil {
		return environment.Frame{}, "", err
	}
	return resp.toFrame(), resp.GUID, nil
}

// Session is one game session against the real API: a specific game_id and
// scorecard card_id, plus the session guid the server hands back after
// RESET (required on every action after that). Implements
// environment.Environment.
type Session struct {
	client *Client
	gameID string
	cardID string
	guid   string
}

// NewSession creates a session for one game under an already-open
// scorecard (see Client.OpenScorecard). Call Reset() before any Step().
func NewSession(client *Client, gameID, cardID string) *Session {
	return &Session{client: client, gameID: gameID, cardID: cardID}
}

// Reset implements environment.Environment.
func (s *Session) Reset() (environment.Frame, error) {
	frame, guid, err := s.client.cmd(context.Background(), "RESET", actionInput{
		GameID: s.gameID,
		CardID: s.cardID,
	})
	if err != nil {
		return environment.Frame{}, err
	}
	s.guid = guid
	return frame, nil
}

// Step implements environment.Environment.
func (s *Session) Step(action environment.Action) (environment.Frame, error) {
	name, ok := actionNames[action.ID]
	if !ok {
		return environment.Frame{}, fmt.Errorf("remote: unknown action ID %d", action.ID)
	}
	if s.guid == "" {
		return environment.Frame{}, fmt.Errorf("remote: session has no guid yet -- call Reset() first")
	}

	in := actionInput{GameID: s.gameID, CardID: s.cardID, GUID: s.guid}
	if action.IsComplex() {
		x, y := action.X, action.Y
		in.X, in.Y = &x, &y
	}

	frame, guid, err := s.client.cmd(context.Background(), name, in)
	if err != nil {
		return environment.Frame{}, err
	}
	s.guid = guid
	return frame, nil
}

// do sends one JSON request and decodes the JSON response into out (if
// out is non-nil). Auth and content headers match main.py's HEADERS
// exactly: X-API-Key and Accept: application/json.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("remote: marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	fullURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return fmt.Errorf("remote: build request: %w", err)
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("remote: %s %s: %w", method, fullURL, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("remote: read response body from %s %s: %w", method, fullURL, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("remote: %s %s: HTTP %d: %s", method, fullURL, resp.StatusCode, string(data))
	}

	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("remote: decode response from %s %s: %w", method, fullURL, err)
		}
	}
	return nil
}
