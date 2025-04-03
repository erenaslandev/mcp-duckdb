package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/erenaslandev/mcp-duckdb/duckdb"
)

type ToolHandler interface {
	HandleQueryTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

type DefaultToolHandler struct {
	client duckdb.Client
}

func NewToolHandler(client duckdb.Client) ToolHandler {
	return &DefaultToolHandler{client: client}
}

func (h *DefaultToolHandler) HandleQueryTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.Params.Arguments
	query, ok1 := arguments["query"].(string)
	limit := 100

	if !ok1 {
		return mcp.NewToolResultError("You must specify the 'query' parameter"), nil
	}

	if limitVal, ok := arguments["limit"].(float64); ok {
		limit = int(limitVal)
	}

	results, err := h.client.QueryData(ctx, query, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("can not execute query: %s", err)), nil
	}

	if len(results.Columns) == 0 {
		return mcp.NewToolResultText("request completed, no results."), nil
	}

	jsonBytes, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("can not format results: %s", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func RegisterTools(mcpServer *server.MCPServer, handler ToolHandler) {
	mcpServer.AddTool(mcp.NewTool("query",
		mcp.WithDescription("SQL request in DuckDB"),
		mcp.WithString("query", mcp.Description("SQL query to execute"), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("Maximum number of strings to be returned (default is 100)")),
	), handler.HandleQueryTool)
}
