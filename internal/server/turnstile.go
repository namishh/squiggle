package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type TTResponse struct {
	Success bool     `json:"success"`
	Errors  []string `json:"error-codes"`
}

func (s *Server) ttverify(ctx context.Context, token string, remoteIp string) (bool, error) {
	form := url.Values{
		"secret":   {s.cfg.TurnstileSecretKey()},
		"response": {token},
		"remoteip": {remoteIp},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://challenges.cloudflare.com/turnstile/v0/siteverify", strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result TTResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	if !result.Success {
		return false, fmt.Errorf("turnstile failed: %v", result.Errors)
	}

	return true, nil
}
