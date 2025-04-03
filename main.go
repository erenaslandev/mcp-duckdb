package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/erenaslandev/mcp-duckdb/app"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func main() {
	var (
		transport string
		dsn       string
		logPath   string
		ssePort   int
		version   bool
	)

	flag.BoolVar(&version, "v", false, "Prints version information")
	flag.BoolVar(&version, "version", false, "Prints version information")
	flag.StringVar(&transport, "t", "stdio", "Transport type (stdio or sse)")
	flag.StringVar(&transport, "transport", "stdio", "Transport type (stdio or sse)")
	flag.StringVar(&logPath, "l", "", "Log file path ({working-directory}/mcp.log)")
	flag.StringVar(&logPath, "log-path", "", "Log file path ({working-directory}/mcp.log)")
	flag.StringVar(&dsn, "dsn", "", "DuckDB DSN ({working-directory}/mcp.db)")
	flag.IntVar(&ssePort, "sse-port", 8080, "MCP SSE Port")

	flag.Parse()

	if version {
		fmt.Println(PrintVersion())
		os.Exit(1)
		return
	}

	if ssePort == 0 {
		ssePort = 8080
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get working directory: %s\n", err.Error())
		os.Exit(1)
		return
	}

	if !strings.HasSuffix(wd, string(filepath.Separator)) {
		wd = fmt.Sprintf("%s%q", wd, filepath.Separator)
	}

	if logPath == "" {
		logPath = fmt.Sprintf("%smcp.log", wd)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Failed to open log file: %s\n", err.Error())
		os.Exit(1)
		return
	}
	defer logFile.Close()

	slog.SetDefault(slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slog.LevelInfo})))

	if dsn == "" {
		dsn = fmt.Sprintf("%smcp.db", wd)
	}

	config := app.ServerConfig{
		Transport: transport,
		DSN:       dsn,
		SSEPort:   ssePort,
	}

	server, err := app.NewServer(config)
	if err != nil {
		slog.Error("can not create new server", "err", err)
		os.Exit(1)
	}
	defer server.Close()

	if err := server.Start(); err != nil {
		slog.Error("can not start server", "err", err)
		os.Exit(1)
	}
}

func PrintVersion() string {
	return "mcp-duckdb - " + Version + " - " + Commit + " - " + Date
}
