// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 ImBIOS — licensed under Apache-2.0, see LICENSE-APACHE.md.

// Package api is the shared control-plane HTTP client. The CLI, the MCP
// server, and headless integrations all speak through Conn so the wire
// contract has exactly one implementation.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Conn is an authenticated control-plane connection.
type Conn struct {
	URL   string
	Token string
	HTTP  *http.Client
}

// New builds a Conn with a 60s HTTP timeout.
func New(url, token string) Conn {
	return Conn{URL: strings.TrimSuffix(url, "/"), Token: token, HTTP: &http.Client{Timeout: 60 * time.Second}}
}

// APIError is a non-2xx control-plane response.
type APIError struct {
	Status int
	Msg    string
}

func (e *APIError) Error() string { return fmt.Sprintf("sidekick: HTTP %d: %s", e.Status, e.Msg) }

func (c Conn) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

// Call performs an authenticated JSON request, decoding JSON into out.
func (c Conn) Call(method, path string, body any, out any) error {
	_, raw, err := c.roundTrip(method, path, body)
	if err != nil {
		return err
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("bad response from control-plane: %w", err)
		}
	}
	return nil
}

// CallBytes performs an authenticated request and returns raw bytes
// (PNG screenshots, MP4 recordings).
func (c Conn) CallBytes(method, path string, body any) ([]byte, error) {
	_, raw, err := c.roundTrip(method, path, body)
	return raw, err
}

func (c Conn) roundTrip(method, path string, body any) (int, []byte, error) {
	if c.Token == "" {
		return 0, nil, fmt.Errorf("not logged in: run `sidekick login` or set SIDEKICK_TOKEN")
	}
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, c.URL+path, rdr)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client().Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("control-plane unreachable at %s: %w", c.URL, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		var parsed map[string]string
		if json.Unmarshal(raw, &parsed) == nil && parsed["error"] != "" {
			msg = parsed["error"]
		}
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return resp.StatusCode, nil, &APIError{Status: resp.StatusCode, Msg: msg}
	}
	return resp.StatusCode, raw, nil
}
