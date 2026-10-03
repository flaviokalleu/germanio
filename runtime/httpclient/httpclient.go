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
	"strconv"
	"strings"
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
			permitirLocal := os.Getenv("GERMANIO_PERMITIR_REDE_LOCAL") == "1"
			if !permitirLocal && nomeLocal(host) {
				return nil, ErrRedeLocal
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			if len(ips) == 0 {
				return nil, fmt.Errorf("sem endereço para %s", host)
			}
			if !permitirLocal {
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

// nomeLocal recognizes, before any resolution, the names and the numeric
// spellings that some resolvers turn into a local address and others
// reject: localhost (RFC 6761) and the short IPv4 forms (127.1,
// 2130706433, 0x7f.1, 0177.0.0.1). The answer does not depend on the
// system resolver.
func nomeLocal(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true
	}
	if ip := ipv4Numerico(h); ip != nil {
		return local(ip)
	}
	return false
}

// ipv4Numerico reads the inet_aton forms of an IPv4 address: one to four
// parts in decimal, octal (leading 0) or hexadecimal (0x), the last part
// filling the remaining bytes. Anything else is not a number.
func ipv4Numerico(h string) net.IP {
	partes := strings.Split(h, ".")
	if len(partes) > 4 {
		return nil
	}
	nums := make([]uint64, len(partes))
	for i, p := range partes {
		base := 10
		switch {
		case strings.HasPrefix(p, "0x"):
			p, base = p[2:], 16
		case len(p) > 1 && p[0] == '0':
			p, base = p[1:], 8
		}
		n, err := strconv.ParseUint(p, base, 32)
		if err != nil {
			return nil
		}
		nums[i] = n
	}
	var v uint64
	for i, n := range nums[:len(nums)-1] {
		if n > 0xff {
			return nil
		}
		v |= n << (24 - 8*uint(i))
	}
	ultimo := nums[len(nums)-1]
	if ultimo >= 1<<(8*uint(5-len(nums))) {
		return nil
	}
	v |= ultimo
	return net.IPv4(byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

// LocalAllowed reports whether GERMANIO_PERMITIR_REDE_LOCAL=1 lets
// outgoing connections reach the server's own networks (development only).
func LocalAllowed() bool { return os.Getenv("GERMANIO_PERMITIR_REDE_LOCAL") == "1" }

// LocalHost reports whether host — a name or an address literal, with or
// without brackets — is recognisably local before any resolution. It gives
// an early, clear refusal when an address is saved; the dialer of Transport
// still checks every resolved address when connecting.
func LocalHost(host string) bool {
	h := strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	if nomeLocal(h) {
		return true
	}
	if ip := net.ParseIP(h); ip != nil {
		return local(ip)
	}
	return false
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
