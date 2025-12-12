package mcps

import "net/http"

type ServerType string

const (
	ServerTypeStdio ServerType = "stdio"
	ServerTypeSSE   ServerType = "sse"
)

type Config struct {
	Servers map[string]*Server `json:"mcpServers"`
}

type Server struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
	Type    ServerType        `json:"type,omitempty"`    // "stdio" (default) or "sse"
	URL     string            `json:"url,omitempty"`     // For SSE
	Headers map[string]string `json:"headers,omitempty"` // For SSE
}

type headerTransport struct {
	Transport http.RoundTripper
	Headers   map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}
	return t.Transport.RoundTrip(req)
}
