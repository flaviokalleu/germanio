package httpclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a simple HTTP client wrapper for Germanio.
type Client struct {
	http *http.Client
}

// Novo creates a new HTTP client with sensible defaults.
func Novo() *Client {
	return &Client{
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Chamar makes an HTTP request and returns the response body.
func (c *Client) Chamar(method, url string, body []byte) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Germanio/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição %s %s: %w", method, url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %w", err)
	}

	if resp.StatusCode >= 400 {
		return respBody, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
