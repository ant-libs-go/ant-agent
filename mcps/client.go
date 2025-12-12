package mcps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sashabaranov/go-openai"
)

type McpClient struct {
	cfg      *Config
	sessions map[string]*mcp.ClientSession
}

// path: config path to mcp.json
func NewMcpClient(path string) (r *McpClient, err error) {
	r = &McpClient{
		sessions: make(map[string]*mcp.ClientSession),
	}

	if err = r.parseConfig(path); err != nil {
		return
	}

	for name, server := range r.cfg.Servers {
		if err = r.connect(name, server); err != nil {
			err = fmt.Errorf("failed to connect to server %s: %w", name, err)
			return
		}
	}
	return
}

func (this *McpClient) parseConfig(path string) (err error) {
	var b []byte
	if b, err = os.ReadFile(path); err != nil {
		err = fmt.Errorf("failed to read config file: %w", err)
		return
	}

	var config *Config
	if err = json.Unmarshal(b, &config); err != nil {
		err = fmt.Errorf("failed to parse config file: %w", err)
		return
	}

	this.cfg = config
	return
}

func (this *McpClient) connect(name string, server *Server) (err error) {
	var transport mcp.Transport

	switch server.Type {
	case ServerTypeSSE:
		transport = this.buildSSETransport(server)
	case ServerTypeStdio:
		transport = this.buildStdioTransport(server)
	default:
		err = fmt.Errorf("unknown server type: %s", server.Type)
		return
	}

	cli := mcp.NewClient(&mcp.Implementation{
		Name:    "ant-agent",
		Version: "0.1.0",
	}, nil)

	var session *mcp.ClientSession
	if session, err = cli.Connect(context.Background(), transport, nil); err != nil {
		err = fmt.Errorf("failed to connect to server: %w", err)
		return
	}

	this.sessions[name] = session
	return
}

func (this *McpClient) buildSSETransport(server *Server) (r mcp.Transport) {
	transport := &mcp.SSEClientTransport{
		Endpoint: server.URL,
	}
	if len(server.Headers) > 0 {
		transport.HTTPClient = &http.Client{
			Transport: &headerTransport{
				Transport: http.DefaultTransport,
				Headers:   server.Headers,
			},
		}
	}
	r = transport
	return
}

func (this *McpClient) buildStdioTransport(server *Server) (r mcp.Transport) {
	cmd := exec.Command(server.Command, server.Args...)
	cmd.Env = os.Environ()
	for k, v := range server.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	cmd.Stderr = os.Stderr // 捕获 stderr 以进行调试

	r = &mcp.CommandTransport{
		Command: cmd,
	}
	return
}

func (this *McpClient) Close() (err error) {
	var errs []error
	for _, session := range this.sessions {
		if err = session.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		err = fmt.Errorf("failed to close some sessions: %v", errs)
		return
	}
	return
}

func (this *McpClient) GetTools() (r []openai.Tool) {
	for name, session := range this.sessions {
		listToolsResult, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list tools from server %s: %v\n", name, err)
			continue
		}

		for _, tool := range listToolsResult.Tools {
			openaiTool := openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        fmt.Sprintf("%s__%s", name, tool.Name),
					Description: tool.Description,
					Parameters:  tool.InputSchema,
				},
			}
			r = append(r, openaiTool)
		}
	}
	return
}

func (this *McpClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (r interface{}, err error) {
	var serverName, toolName string

	if serverName, toolName, err = this.parseToolName(name); err != nil {
		err = fmt.Errorf("failed to parse tool name: %w", err)
		return
	}

	var ok bool
	var session *mcp.ClientSession
	if session, ok = this.sessions[serverName]; !ok {
		err = fmt.Errorf("server %s not found", serverName)
		return
	}

	var result *mcp.CallToolResult
	if result, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	}); err != nil {
		err = fmt.Errorf("failed to call tool: %w", err)
		return
	}

	r = result
	return
}

func (this *McpClient) parseToolName(name string) (serverName string, toolName string, err error) {
	parts := strings.Split(name, "__")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid tool name format: %s", name)
	}
	return parts[0], parts[1], nil
}
