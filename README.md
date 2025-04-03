# ClickHouse MCP Server

[![Go Version](https://img.shields.io/github/go-mod/go-version/erenaslandev/mcp-duckdb)](https://go.dev)
[![License](https://img.shields.io/github/license/erenaslandev/mcp-duckdb)](LICENSE)

MCP-compatible server for interacting with DuckDB databases.

## Features

- Executing SQL queries and getting results
- Support for different transports (stdio and SSE)

## Usage

### Run

Running via stdio (by default):

```bash
/path/mcp-duckdb -l /path/mcp.log -dsn /path/mcp.db
```

Running via SSE:

```bash
/path/mcp-duckdb -t sse -l /path/mcp.log -dsn /path/mcp.db
```

### MCP Client Configuration

```json
{
	"mcpServers": {
		"mcp-duckdb": {
			"command": "/path/mcp-duckdb",
			"args": [
				"-l",
				"/path/mcp.log",
				"-dsn",
				"/path/mcp.db"
			],
			"disabled": false,
			"alwaysAllow": []
		}
	}
}
```

### MCP Client Configuration with SSE

```json
{
 "mcpServers": {
   "mcp-duckdb": {
     "url": "http://localhost:8080/sse"
   }
 }
}
```

## License

MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create a branch for your changes
3. Make your changes and create a pull request

## Contact

Create an issue in the repository to report problems or suggest improvements.

## Credits

- [@Headcrab](https://github.com/Headcrab)'s [clickhouse-mcp](https://github.com/Headcrab/clickhouse-mcp) server implementation.