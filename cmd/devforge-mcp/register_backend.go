// cmd/devforge-mcp/register_backend.go registers the backend utility MCP tools.
package main

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"dev-forge-mcp/internal/tools/backend"
)

// registerBackendTools registers all backend utility tools with the MCP server.
func registerBackendTools(s *mcpserver.MCPServer, _ *mcpApp) {
	// ── backend_sql_format ────────────────────────────────────────
	s.AddTool(mcp.NewTool("backend_sql_format",
		mcp.WithDescription("Format and lint a SQL statement with configurable indentation, keyword casing, and dialect-aware warnings."),
		mcp.WithString("sql", mcp.Required(), mcp.Description("SQL statement to format")),
		mcp.WithString("dialect", mcp.Description("SQL dialect: postgresql | mysql | sqlite | generic (default: generic)")),
		mcp.WithString("indent", mcp.Description("Indentation string (default: two spaces)")),
		mcp.WithBoolean("uppercase_keywords", mcp.Description("Uppercase SQL keywords (default: true)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := backend.SQLFormatInput{
			SQL:              mcp.ParseString(req, "sql", ""),
			Dialect:          mcp.ParseString(req, "dialect", "generic"),
			Indent:           mcp.ParseString(req, "indent", "  "),
			UppercaseKeyword: mcp.ParseBoolean(req, "uppercase_keywords", true),
		}
		return mcp.NewToolResultText(backend.SQLFormat(ctx, input)), nil
	})

	// ── backend_conn_string ───────────────────────────────────────
	s.AddTool(mcp.NewTool("backend_conn_string",
		mcp.WithDescription("Build or parse a database connection string (DSN) for PostgreSQL, MySQL, MongoDB, or Redis."),
		mcp.WithString("operation", mcp.Required(), mcp.Description("build | parse")),
		mcp.WithString("db_type", mcp.Required(), mcp.Description("postgresql | mysql | mongodb | redis")),
		mcp.WithString("connection_string", mcp.Description("Connection string to parse (required for parse operation)")),
		mcp.WithString("host", mcp.Description("Database host")),
		mcp.WithNumber("port", mcp.Description("Database port")),
		mcp.WithString("database", mcp.Description("Database name")),
		mcp.WithString("username", mcp.Description("Username")),
		mcp.WithString("password", mcp.Description("Password")),
		mcp.WithObject("options", mcp.Description("Extra key/value connection parameters")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := backend.ConnStringInput{
			Operation:        mcp.ParseString(req, "operation", "build"),
			DBType:           mcp.ParseString(req, "db_type", ""),
			ConnectionString: mcp.ParseString(req, "connection_string", ""),
			Host:             mcp.ParseString(req, "host", ""),
			Port:             mcp.ParseInt(req, "port", 0),
			Database:         mcp.ParseString(req, "database", ""),
			Username:         mcp.ParseString(req, "username", ""),
			Password:         mcp.ParseString(req, "password", ""),
		}
		if optsRaw, ok := args["options"].(map[string]interface{}); ok {
			input.Options = make(map[string]string)
			for k, v := range optsRaw {
				input.Options[k] = stringify(v)
			}
		}
		return mcp.NewToolResultText(backend.ConnString(ctx, input)), nil
	})

	// ── backend_log_parse ─────────────────────────────────────────
	s.AddTool(mcp.NewTool("backend_log_parse",
		mcp.WithDescription("Parse multiline log content (JSON, NDJSON, Apache, Nginx), filter by field values and time range, return matching entries."),
		mcp.WithString("log", mcp.Required(), mcp.Description("Multiline log content to parse")),
		mcp.WithString("format", mcp.Description("Log format: json | ndjson | apache | nginx | auto (default: auto)")),
		mcp.WithObject("filter", mcp.Description("Key/value pairs to filter log entries by")),
		mcp.WithString("start_time", mcp.Description("ISO8601 start time for time-range filtering")),
		mcp.WithString("end_time", mcp.Description("ISO8601 end time for time-range filtering")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of entries to return (default: 100)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := backend.LogParseInput{
			Log:       mcp.ParseString(req, "log", ""),
			Format:    mcp.ParseString(req, "format", "auto"),
			StartTime: mcp.ParseString(req, "start_time", ""),
			EndTime:   mcp.ParseString(req, "end_time", ""),
			Limit:     mcp.ParseInt(req, "limit", 100),
		}
		if filterRaw, ok := args["filter"].(map[string]interface{}); ok {
			input.Filter = filterRaw
		}
		return mcp.NewToolResultText(backend.LogParse(ctx, input)), nil
	})

	// ── backend_env_inspect ───────────────────────────────────────
	s.AddTool(mcp.NewTool("backend_env_inspect",
		mcp.WithDescription("Validate a .env file against a schema or generate a .env.example from it."),
		mcp.WithString("env_content", mcp.Required(), mcp.Description("Contents of the .env file")),
		mcp.WithString("schema", mcp.Description("JSON object mapping env keys to {required, description, pattern} specs")),
		mcp.WithString("operation", mcp.Description("validate | generate_example (default: validate)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := backend.EnvInspectInput{
			EnvContent: mcp.ParseString(req, "env_content", ""),
			Schema:     mcp.ParseString(req, "schema", ""),
			Operation:  mcp.ParseString(req, "operation", "validate"),
		}
		return mcp.NewToolResultText(backend.EnvInspect(ctx, input)), nil
	})

	// ── backend_mq_payload ────────────────────────────────────────
	s.AddTool(mcp.NewTool("backend_mq_payload",
		mcp.WithDescription("Build a message queue payload envelope for Kafka, RabbitMQ, or SQS. Pure serialization — no broker connections made."),
		mcp.WithString("broker", mcp.Required(), mcp.Description("Message broker: kafka | rabbitmq | sqs")),
		mcp.WithString("operation", mcp.Description("build | serialize | format (default: build)")),
		mcp.WithString("topic", mcp.Description("Kafka topic / RabbitMQ routing key / SQS queue name or URL")),
		mcp.WithString("payload", mcp.Description("JSON body of the message")),
		mcp.WithObject("headers", mcp.Description("Message headers as key/value pairs")),
		mcp.WithObject("options", mcp.Description("Broker-specific options (e.g. key, partition for Kafka; exchange for RabbitMQ; queue_url, message_group_id for SQS)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := backend.MQPayloadInput{
			Broker:    mcp.ParseString(req, "broker", ""),
			Operation: mcp.ParseString(req, "operation", "build"),
			Topic:     mcp.ParseString(req, "topic", ""),
			Payload:   mcp.ParseString(req, "payload", ""),
		}
		if headersRaw, ok := args["headers"].(map[string]interface{}); ok {
			input.Headers = make(map[string]string)
			for k, v := range headersRaw {
				input.Headers[k] = stringify(v)
			}
		}
		if optsRaw, ok := args["options"].(map[string]interface{}); ok {
			input.Options = optsRaw
		}
		return mcp.NewToolResultText(backend.MQPayload(ctx, input)), nil
	})

	// ── backend_cidr_subnet ───────────────────────────────────────
	s.AddTool(mcp.NewTool("backend_cidr_subnet",
		mcp.WithDescription("Calculate IPv4 CIDR subnet details including network, broadcast, usable range, and optional host list."),
		mcp.WithString("cidr", mcp.Required(), mcp.Description("IPv4 CIDR block, e.g. 10.0.0.0/24")),
		mcp.WithBoolean("include_all", mcp.Description("Include usable host IP list in the response (default: true)")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of host IPs to return when include_all=true (default: 256)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := backend.CIDRSubnetInput{
			CIDR:       mcp.ParseString(req, "cidr", ""),
			IncludeAll: mcp.ParseBoolean(req, "include_all", true),
			Limit:      mcp.ParseInt(req, "limit", 256),
		}
		return mcp.NewToolResultText(backend.CIDRSubnet(ctx, input)), nil
	})
}

// stringify converts an arbitrary interface value to its string representation.
func stringify(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}
