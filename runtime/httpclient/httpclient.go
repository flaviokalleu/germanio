package httpclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

// ErrRedeLocal: the destination resolves to the server's own networks.
var ErrRedeLocal = errors.New("endereço de rede local bloqueado (defina GERMANIO_PERMITIR_REDE_LOCAL=1 para permitir)")

// maxResposta is the most a response body may carry.
const maxResposta = 10 << 20

// Transport is the one way Germanio talks to other servers. It refuses
// loopback, private, link-local and unspecified destinations, checking the
// resolved address when connecting, so no spelling of an address, DNS
// rebinding or redirect reaches the server's own network (SSRF).
// GERMANIO_PERMITIR_REDE_LOCAL=1 allows them (development only).
func Transport() *http.Transport {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			if len(ips) == 0 {
				return nil, fmt.Errorf("sem endereço para %s", host)
			}
			if os.Getenv("GERMANIO_PERMITIR_REDE_LOCAL") != "1" {
				for _, ip := range ips {
					if local(ip.IP) {
						return nil, ErrRedeLocal
					}
				}
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
		TLSHandshakeTimeout: 5 * time.Second,
	}
}

func local(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		return v4[0] == 0 || (v4[0] == 100 && v4[1]&0xc0 == 64) // 0.0.0.0/8, shared address space 100.64/10
	}
	return false
}

// Client is a simple HTTP client wrapper for Germanio.
type Client struct {
	http *http.Client
}

// Novo creates a client that only reaches public addresses (see Transport),
// follows at most 5 redirects (each one checked again) and times out.
func Novo() *Client {
	return &Client{
		http: &http.Client{
			Transport: Transport(),
			Timeout:   30 * time.Second,
			CheckRedirect: func(_ *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errors.New("redirecionamentos demais")
				}
				return nil
			},
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

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResposta+1))
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %w", err)
	}
	if len(respBody) > maxResposta {
		return nil, fmt.Errorf("resposta maior que %d MB", maxResposta>>20)
	}

	if resp.StatusCode >= 400 {
		return respBody, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
