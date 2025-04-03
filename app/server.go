package app

import (
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/server"

	"github.com/erenaslandev/mcp-duckdb/duckdb"
)

type ServerConfig struct {
	Transport string
	DSN       string
	SSEPort   int
}

type Server struct {
	mcpServer *server.MCPServer
	dbClient  duckdb.Client
	config    ServerConfig
	tools     ToolHandler
}

func NewServer(config ServerConfig) (*Server, error) {
	server := &Server{config: config}

	client, err := duckdb.NewClient(duckdb.Config{DSN: server.config.DSN})
	if err != nil {
		return nil, err
	}

	server.dbClient = client

	server.tools = NewToolHandler(server.dbClient)

	server.mcpServer = server.createMCPServer()

	RegisterTools(server.mcpServer, server.tools)

	return server, nil
}

func (s *Server) createMCPServer() *server.MCPServer {
	return server.NewMCPServer(
		"mcp-duckdb",
		"1.0.0",
		server.WithLogging(),
	)
}

func (s *Server) Start() error {
	if s.config.Transport == "sse" {
		sseServer := server.NewSSEServer(s.mcpServer, server.WithBaseURL(fmt.Sprintf("http://localhost:%d", s.config.SSEPort)))
		slog.Info(fmt.Sprintf("SSE server running on %d", s.config.SSEPort))
		if err := sseServer.Start(fmt.Sprintf("0.0.0.0:%d", s.config.SSEPort)); err != nil {
			return fmt.Errorf("can not start SSE server: %w", err)
		}
	} else {
		slog.Info("Starting DuckDB MCP server via stdio")
		if err := server.ServeStdio(s.mcpServer); err != nil {
			return fmt.Errorf("can not start stdio server: %w", err)
		}
	}

	return nil
}

func (s *Server) Close() error {
	if s.dbClient == nil {
		return nil
	}

	err := s.dbClient.Close()
	if err != nil {
		return err
	}

	return nil
}
