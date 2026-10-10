//go:build integration

// Integration test: GET /api/v1/auth/me authenticates by API key (Bearer), not
// only the session cookie, so a key holder (e.g. the CLI) can resolve their own
// identity (whoami). Built through handler.Register to exercise the real route
// wiring. Run with: go test -tags=integration ./internal/api/handler/
package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/strelov1/freehire/internal/identity/auth"
)

func TestAuthMe_AuthenticatesByAPIKey(t *testing.T) {
	pool := startPostgres(t)
	ctx := context.Background()

	var userID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email) VALUES ('whoami@example.test') RETURNING id`).Scan(&userID); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	token, hash, prefix, err := auth.GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO api_keys (user_id, name, token_hash, token_prefix) VALUES ($1, 'cli', $2, $3)`,
		userID, hash, prefix); err != nil {
		t.Fatalf("seed api_key: %v", err)
	}

	app := fiber.New(fiber.Config{ErrorHandler: RenderError})
	Register(app, Config{Pool: pool, FrontendOrigin: "http://localhost", JWTSecret: "test-secret", JWTTTL: time.Hour})

	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200 (auth/me must accept an API key)", resp.StatusCode)
	}

	var body struct {
		Data struct {
			Email string `json:"email"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Data.Email != "whoami@example.test" {
		t.Errorf("email = %q, want whoami@example.test", body.Data.Email)
	}
}
