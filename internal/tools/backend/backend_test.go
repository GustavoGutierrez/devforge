package backend_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools/backend"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func jsonMap(t *testing.T, s string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("invalid JSON %q: %v", s, err)
	}
	return m
}

func getString(t *testing.T, m map[string]interface{}, key string) string {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("key %q not found in map %v", key, m)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("key %q is not a string (got %T)", key, v)
	}
	return s
}

func getError(t *testing.T, s string) string {
	t.Helper()
	m := jsonMap(t, s)
	if e, ok := m["error"]; ok {
		if str, ok := e.(string); ok {
			return str
		}
	}
	return ""
}

// ── backend_sql_format ────────────────────────────────────────────────────────

func TestSQLFormat(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		input       backend.SQLFormatInput
		checkResult func(t *testing.T, result string)
		wantError   bool
	}{
		{
			name: "happy path: basic SELECT uppercase keywords",
			input: backend.SQLFormatInput{
				SQL:              "select id, name from users where id = 1",
				Dialect:          "postgresql",
				Indent:           "  ",
				UppercaseKeyword: true,
			},
			checkResult: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				formatted := getString(t, m, "result")
				if !strings.Contains(formatted, "SELECT") {
					t.Errorf("expected uppercase SELECT, got: %s", formatted)
				}
				if !strings.Contains(formatted, "FROM") {
					t.Errorf("expected uppercase FROM, got: %s", formatted)
				}
				if !strings.Contains(formatted, "WHERE") {
					t.Errorf("expected uppercase WHERE, got: %s", formatted)
				}
			},
		},
		{
			name: "happy path: SELECT * warns",
			input: backend.SQLFormatInput{
				SQL:              "SELECT * FROM users",
				UppercaseKeyword: true,
			},
			checkResult: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				warnings, ok := m["warnings"].([]interface{})
				if !ok {
					t.Fatalf("warnings is not an array in %v", m)
				}
				found := false
				for _, w := range warnings {
					if s, ok := w.(string); ok && strings.Contains(s, "SELECT *") {
						found = true
					}
				}
				if !found {
					t.Errorf("expected SELECT * warning, got: %v", warnings)
				}
			},
		},
		{
			name: "happy path: UPDATE without WHERE warns",
			input: backend.SQLFormatInput{
				SQL:              "UPDATE users SET name = 'x'",
				UppercaseKeyword: true,
			},
			checkResult: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				warnings, ok := m["warnings"].([]interface{})
				if !ok {
					t.Fatalf("warnings is not an array")
				}
				found := false
				for _, w := range warnings {
					if s, ok := w.(string); ok && strings.Contains(s, "UPDATE without WHERE") {
						found = true
					}
				}
				if !found {
					t.Errorf("expected UPDATE without WHERE warning, got: %v", warnings)
				}
			},
		},
		{
			name: "error path: empty SQL",
			input: backend.SQLFormatInput{
				SQL: "",
			},
			wantError: true,
		},
		{
			name: "error path: invalid dialect",
			input: backend.SQLFormatInput{
				SQL:     "SELECT 1",
				Dialect: "oracle",
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := backend.SQLFormat(ctx, tc.input)
			if tc.wantError {
				if e := getError(t, got); e == "" {
					t.Errorf("expected error, got: %s", got)
				}
				return
			}
			if tc.checkResult != nil {
				tc.checkResult(t, got)
			}
		})
	}
}

// ── backend_conn_string ───────────────────────────────────────────────────────

func TestConnString(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     backend.ConnStringInput
		checkFn   func(t *testing.T, result string)
		wantError bool
	}{
		{
			name: "happy path: build postgresql DSN",
			input: backend.ConnStringInput{
				Operation: "build",
				DBType:    "postgresql",
				Host:      "localhost",
				Port:      5432,
				Database:  "mydb",
				Username:  "admin",
				Password:  "secret",
				Options:   map[string]string{"sslmode": "require"},
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				cs := getString(t, m, "connection_string")
				if !strings.HasPrefix(cs, "postgresql://") {
					t.Errorf("expected postgresql:// prefix, got: %s", cs)
				}
				if !strings.Contains(cs, "localhost:5432") {
					t.Errorf("expected host:port in DSN, got: %s", cs)
				}
				if !strings.Contains(cs, "mydb") {
					t.Errorf("expected dbname in DSN, got: %s", cs)
				}
			},
		},
		{
			name: "happy path: build mysql DSN",
			input: backend.ConnStringInput{
				Operation: "build",
				DBType:    "mysql",
				Host:      "db.example.com",
				Port:      3306,
				Database:  "appdb",
				Username:  "user",
				Password:  "pass",
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				cs := getString(t, m, "connection_string")
				if !strings.Contains(cs, "@tcp(") {
					t.Errorf("expected MySQL tcp format, got: %s", cs)
				}
				if !strings.Contains(cs, "appdb") {
					t.Errorf("expected dbname, got: %s", cs)
				}
			},
		},
		{
			name: "happy path: build redis DSN",
			input: backend.ConnStringInput{
				Operation: "build",
				DBType:    "redis",
				Host:      "cache.local",
				Port:      6379,
				Password:  "redispass",
				Database:  "1",
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				cs := getString(t, m, "connection_string")
				if !strings.HasPrefix(cs, "redis://") {
					t.Errorf("expected redis:// prefix, got: %s", cs)
				}
			},
		},
		{
			name: "happy path: parse postgresql DSN",
			input: backend.ConnStringInput{
				Operation:        "parse",
				DBType:           "postgresql",
				ConnectionString: "postgresql://admin:secret@localhost:5432/mydb?sslmode=require",
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				if host := getString(t, m, "host"); host != "localhost" {
					t.Errorf("expected host=localhost, got %s", host)
				}
				if db := getString(t, m, "database"); db != "mydb" {
					t.Errorf("expected database=mydb, got %s", db)
				}
				if user := getString(t, m, "username"); user != "admin" {
					t.Errorf("expected username=admin, got %s", user)
				}
			},
		},
		{
			name: "happy path: parse mysql DSN",
			input: backend.ConnStringInput{
				Operation:        "parse",
				DBType:           "mysql",
				ConnectionString: "user:pass@tcp(db.host:3306)/appdb?charset=utf8mb4",
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				if host := getString(t, m, "host"); host != "db.host" {
					t.Errorf("expected host=db.host, got %s", host)
				}
				if db := getString(t, m, "database"); db != "appdb" {
					t.Errorf("expected database=appdb, got %s", db)
				}
			},
		},
		{
			name: "error path: missing db_type",
			input: backend.ConnStringInput{
				Operation: "build",
				DBType:    "oracle",
			},
			wantError: true,
		},
		{
			name: "error path: parse missing connection_string",
			input: backend.ConnStringInput{
				Operation:        "parse",
				DBType:           "postgresql",
				ConnectionString: "",
			},
			wantError: true,
		},
		{
			name: "error path: invalid operation",
			input: backend.ConnStringInput{
				Operation: "delete",
				DBType:    "postgresql",
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := backend.ConnString(ctx, tc.input)
			if tc.wantError {
				if e := getError(t, got); e == "" {
					t.Errorf("expected error, got: %s", got)
				}
				return
			}
			if tc.checkFn != nil {
				tc.checkFn(t, got)
			}
		})
	}
}

// ── backend_log_parse ─────────────────────────────────────────────────────────

func TestLogParse(t *testing.T) {
	ctx := context.Background()

	jsonLogs := `{"level":"info","msg":"server started","time":"2024-01-15T10:00:00Z"}
{"level":"error","msg":"connection failed","time":"2024-01-15T10:01:00Z"}
{"level":"info","msg":"request processed","time":"2024-01-15T10:02:00Z"}`

	apacheLogs := `127.0.0.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 2326 "http://www.example.com/start.html" "Mozilla/4.08"`

	tests := []struct {
		name      string
		input     backend.LogParseInput
		checkFn   func(t *testing.T, result string)
		wantError bool
	}{
		{
			name: "happy path: auto-detect JSON logs",
			input: backend.LogParseInput{
				Log:    jsonLogs,
				Format: "auto",
				Limit:  100,
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				format, ok := m["format_detected"].(string)
				if !ok {
					t.Fatal("format_detected missing")
				}
				if format != "ndjson" && format != "json" {
					t.Errorf("expected ndjson format, got: %s", format)
				}
				entries, ok := m["entries"].([]interface{})
				if !ok || len(entries) == 0 {
					t.Errorf("expected entries, got: %v", m)
				}
				total, ok := m["total"].(float64)
				if !ok || total != 3 {
					t.Errorf("expected total=3, got: %v", m["total"])
				}
			},
		},
		{
			name: "happy path: filter by field",
			input: backend.LogParseInput{
				Log:    jsonLogs,
				Format: "ndjson",
				Filter: map[string]interface{}{"level": "error"},
				Limit:  100,
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				entries, ok := m["entries"].([]interface{})
				if !ok {
					t.Fatal("entries missing")
				}
				if len(entries) != 1 {
					t.Errorf("expected 1 filtered entry, got %d", len(entries))
				}
			},
		},
		{
			name: "happy path: parse apache log",
			input: backend.LogParseInput{
				Log:    apacheLogs,
				Format: "apache",
				Limit:  10,
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				entries, ok := m["entries"].([]interface{})
				if !ok || len(entries) == 0 {
					t.Fatalf("expected entries, got: %v", m)
				}
				entry, ok := entries[0].(map[string]interface{})
				if !ok {
					t.Fatal("entry is not a map")
				}
				if entry["remote_addr"] != "127.0.0.1" {
					t.Errorf("expected remote_addr=127.0.0.1, got %v", entry["remote_addr"])
				}
			},
		},
		{
			name: "happy path: limit results",
			input: backend.LogParseInput{
				Log:   jsonLogs,
				Limit: 2,
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				entries, ok := m["entries"].([]interface{})
				if !ok {
					t.Fatal("entries missing")
				}
				if len(entries) > 2 {
					t.Errorf("expected at most 2 entries, got %d", len(entries))
				}
			},
		},
		{
			name: "error path: empty log",
			input: backend.LogParseInput{
				Log: "",
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := backend.LogParse(ctx, tc.input)
			if tc.wantError {
				if e := getError(t, got); e == "" {
					t.Errorf("expected error, got: %s", got)
				}
				return
			}
			if tc.checkFn != nil {
				tc.checkFn(t, got)
			}
		})
	}
}

// ── backend_env_inspect ───────────────────────────────────────────────────────

func TestEnvInspect(t *testing.T) {
	ctx := context.Background()

	validEnv := `# Database config
DB_HOST=localhost
DB_PORT=5432
DB_NAME=myapp
API_KEY="abc123"
SECRET='mysecret'
`

	schemaJSON := `{
		"DB_HOST": {"required": true, "description": "Database host"},
		"DB_PORT": {"required": true, "description": "Database port", "pattern": "^[0-9]+$"},
		"DB_NAME": {"required": true, "description": "Database name"},
		"API_KEY": {"required": true, "description": "API key"}
	}`

	tests := []struct {
		name      string
		input     backend.EnvInspectInput
		checkFn   func(t *testing.T, result string)
		wantError bool
	}{
		{
			name: "happy path: validate valid env with schema",
			input: backend.EnvInspectInput{
				EnvContent: validEnv,
				Schema:     schemaJSON,
				Operation:  "validate",
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				valid, ok := m["valid"].(bool)
				if !ok {
					t.Fatalf("valid field missing in: %v", m)
				}
				if !valid {
					t.Errorf("expected valid=true, got false. Details: %v", m)
				}
				missing, ok := m["missing_required"].([]interface{})
				if !ok {
					t.Fatal("missing_required missing")
				}
				if len(missing) != 0 {
					t.Errorf("expected no missing required keys, got: %v", missing)
				}
			},
		},
		{
			name: "happy path: detect missing required key",
			input: backend.EnvInspectInput{
				EnvContent: "DB_HOST=localhost\n",
				Schema:     `{"DB_HOST": {"required": true}, "API_KEY": {"required": true}}`,
				Operation:  "validate",
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				valid, _ := m["valid"].(bool)
				if valid {
					t.Error("expected valid=false when required key missing")
				}
				missing, ok := m["missing_required"].([]interface{})
				if !ok || len(missing) == 0 {
					t.Errorf("expected missing_required to contain API_KEY, got: %v", m["missing_required"])
				}
			},
		},
		{
			name: "happy path: generate example",
			input: backend.EnvInspectInput{
				EnvContent: "DB_HOST=localhost\nAPI_KEY=secret\n",
				Schema:     `{"DB_HOST": {"required": true, "description": "Database hostname"}}`,
				Operation:  "generate_example",
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				example := getString(t, m, "example")
				if !strings.Contains(example, "DB_HOST=") {
					t.Errorf("expected DB_HOST in example, got: %s", example)
				}
			},
		},
		{
			name: "happy path: detect invalid format",
			input: backend.EnvInspectInput{
				EnvContent: "DB_PORT=not_a_port\n",
				Schema:     `{"DB_PORT": {"required": true, "pattern": "^[0-9]+$"}}`,
				Operation:  "validate",
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				valid, _ := m["valid"].(bool)
				if valid {
					t.Error("expected valid=false for invalid format")
				}
				invalidFmt, ok := m["invalid_format"].([]interface{})
				if !ok || len(invalidFmt) == 0 {
					t.Errorf("expected invalid_format entries, got: %v", m["invalid_format"])
				}
			},
		},
		{
			name: "error path: empty env_content",
			input: backend.EnvInspectInput{
				EnvContent: "",
				Operation:  "validate",
			},
			wantError: true,
		},
		{
			name: "error path: invalid operation",
			input: backend.EnvInspectInput{
				EnvContent: "KEY=val\n",
				Operation:  "export",
			},
			wantError: true,
		},
		{
			name: "error path: invalid schema JSON",
			input: backend.EnvInspectInput{
				EnvContent: "KEY=val\n",
				Schema:     "not-json",
				Operation:  "validate",
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := backend.EnvInspect(ctx, tc.input)
			if tc.wantError {
				if e := getError(t, got); e == "" {
					t.Errorf("expected error, got: %s", got)
				}
				return
			}
			if tc.checkFn != nil {
				tc.checkFn(t, got)
			}
		})
	}
}

// ── backend_mq_payload ────────────────────────────────────────────────────────

func TestMQPayload(t *testing.T) {
	ctx := context.Background()

	samplePayload := `{"event": "user.created", "user_id": 42}`

	tests := []struct {
		name      string
		input     backend.MQPayloadInput
		checkFn   func(t *testing.T, result string)
		wantError bool
	}{
		{
			name: "happy path: kafka envelope",
			input: backend.MQPayloadInput{
				Broker:    "kafka",
				Operation: "build",
				Topic:     "user-events",
				Payload:   samplePayload,
				Headers:   map[string]string{"content-type": "application/json"},
				Options:   map[string]interface{}{"key": "user-42", "partition": float64(2)},
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				if broker := getString(t, m, "broker"); broker != "kafka" {
					t.Errorf("expected broker=kafka, got %s", broker)
				}
				if _, ok := m["serialized"]; !ok {
					t.Error("serialized field missing")
				}
				envelope, ok := m["envelope"].(map[string]interface{})
				if !ok {
					t.Fatal("envelope is not a map")
				}
				if topic, _ := envelope["topic"].(string); topic != "user-events" {
					t.Errorf("expected topic=user-events, got %s", topic)
				}
				if key, _ := envelope["key"].(string); key != "user-42" {
					t.Errorf("expected key=user-42, got %s", key)
				}
			},
		},
		{
			name: "happy path: rabbitmq envelope",
			input: backend.MQPayloadInput{
				Broker:    "rabbitmq",
				Operation: "build",
				Topic:     "user.created",
				Payload:   samplePayload,
				Options:   map[string]interface{}{"exchange": "events", "delivery_mode": float64(2)},
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				envelope, ok := m["envelope"].(map[string]interface{})
				if !ok {
					t.Fatal("envelope missing")
				}
				if rk, _ := envelope["routing_key"].(string); rk != "user.created" {
					t.Errorf("expected routing_key=user.created, got %s", rk)
				}
				if ex, _ := envelope["exchange"].(string); ex != "events" {
					t.Errorf("expected exchange=events, got %s", ex)
				}
				props, ok := envelope["properties"].(map[string]interface{})
				if !ok {
					t.Fatal("properties missing in RabbitMQ envelope")
				}
				if ct, _ := props["content_type"].(string); ct != "application/json" {
					t.Errorf("expected content_type=application/json, got %s", ct)
				}
			},
		},
		{
			name: "happy path: sqs envelope",
			input: backend.MQPayloadInput{
				Broker:    "sqs",
				Operation: "build",
				Topic:     "https://sqs.us-east-1.amazonaws.com/123/my-queue",
				Payload:   samplePayload,
				Headers:   map[string]string{"source": "backend"},
				Options:   map[string]interface{}{"message_group_id": "group-1"},
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				envelope, ok := m["envelope"].(map[string]interface{})
				if !ok {
					t.Fatal("envelope missing")
				}
				if _, ok := envelope["QueueUrl"]; !ok {
					t.Error("QueueUrl missing in SQS envelope")
				}
				if _, ok := envelope["MessageBody"]; !ok {
					t.Error("MessageBody missing in SQS envelope")
				}
				if _, ok := envelope["MessageGroupId"]; !ok {
					t.Error("MessageGroupId missing in SQS envelope")
				}
			},
		},
		{
			name: "happy path: serialized is valid JSON",
			input: backend.MQPayloadInput{
				Broker:  "kafka",
				Topic:   "test",
				Payload: `{"key": "value"}`,
			},
			checkFn: func(t *testing.T, result string) {
				m := jsonMap(t, result)
				serialized, ok := m["serialized"].(string)
				if !ok || serialized == "" {
					t.Fatal("serialized is empty or missing")
				}
				var v interface{}
				if err := json.Unmarshal([]byte(serialized), &v); err != nil {
					t.Errorf("serialized is not valid JSON: %v", err)
				}
			},
		},
		{
			name: "error path: invalid broker",
			input: backend.MQPayloadInput{
				Broker: "activemq",
				Topic:  "test",
			},
			wantError: true,
		},
		{
			name: "error path: invalid operation",
			input: backend.MQPayloadInput{
				Broker:    "kafka",
				Operation: "consume",
				Topic:     "test",
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := backend.MQPayload(ctx, tc.input)
			if tc.wantError {
				if e := getError(t, got); e == "" {
					t.Errorf("expected error, got: %s", got)
				}
				return
			}
			if tc.checkFn != nil {
				tc.checkFn(t, got)
			}
		})
	}
}
