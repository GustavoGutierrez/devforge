// Package backend implements MCP tools for backend development utilities.
// Tools: backend_sql_format, backend_conn_string, backend_log_parse,
// backend_env_inspect, backend_mq_payload.
package backend

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// errResult returns a JSON-encoded error response.
func errResult(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}

// resultJSON marshals v to JSON or returns an error JSON.
func resultJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return errResult("marshal failed: " + err.Error())
	}
	return string(b)
}

// ─── backend_sql_format ──────────────────────────────────────────────────────

// SQLFormatInput is the input schema for the backend_sql_format tool.
type SQLFormatInput struct {
	SQL              string `json:"sql"`
	Dialect          string `json:"dialect"`            // postgresql | mysql | sqlite | generic
	Indent           string `json:"indent"`             // default "  "
	UppercaseKeyword bool   `json:"uppercase_keywords"` // default true
}

// SQLFormatOutput is the output for the backend_sql_format tool.
type SQLFormatOutput struct {
	Result   string   `json:"result"`
	Warnings []string `json:"warnings"`
}

// SQL keywords to recognize and reformat.
var sqlKeywords = []string{
	"SELECT", "FROM", "WHERE", "JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "FULL",
	"ON", "AND", "OR", "NOT", "IN", "EXISTS", "BETWEEN", "LIKE", "IS", "NULL",
	"INSERT", "INTO", "VALUES", "UPDATE", "SET", "DELETE", "CREATE", "TABLE",
	"ALTER", "DROP", "INDEX", "VIEW", "DATABASE", "SCHEMA", "CONSTRAINT",
	"PRIMARY", "KEY", "FOREIGN", "REFERENCES", "UNIQUE", "CHECK", "DEFAULT",
	"ORDER", "BY", "GROUP", "HAVING", "LIMIT", "OFFSET", "UNION", "ALL",
	"CASE", "WHEN", "THEN", "ELSE", "END", "AS", "DISTINCT", "ASC", "DESC",
	"WITH", "RECURSIVE", "RETURNING", "BEGIN", "COMMIT", "ROLLBACK", "TRANSACTION",
	"COUNT", "SUM", "AVG", "MIN", "MAX", "COALESCE", "NULLIF", "CAST",
	"CROSS", "NATURAL", "USING", "EXCEPT", "INTERSECT",
}

// sqlKeywordSet is a set of uppercase SQL keywords for O(1) lookup.
var sqlKeywordSet map[string]bool

func init() {
	sqlKeywordSet = make(map[string]bool, len(sqlKeywords))
	for _, kw := range sqlKeywords {
		sqlKeywordSet[kw] = true
	}
}

// tokenType classifies a SQL token.
type tokenType int

const (
	tokenWord     tokenType = iota // identifier or keyword
	tokenNumber                    // numeric literal
	tokenString                    // quoted string literal
	tokenComment                   // line or block comment
	tokenOperator                  // operator or punctuation
	tokenWhitespace
)

// sqlToken is a tokenized piece of SQL.
type sqlToken struct {
	typ  tokenType
	text string
}

// tokenizeSQL splits raw SQL into a slice of tokens.
func tokenizeSQL(sql string) []sqlToken {
	var tokens []sqlToken
	i := 0
	n := len(sql)
	for i < n {
		ch := sql[i]

		// Whitespace
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			j := i
			for j < n && (sql[j] == ' ' || sql[j] == '\t' || sql[j] == '\n' || sql[j] == '\r') {
				j++
			}
			tokens = append(tokens, sqlToken{typ: tokenWhitespace, text: sql[i:j]})
			i = j
			continue
		}

		// Line comment --
		if i+1 < n && sql[i] == '-' && sql[i+1] == '-' {
			j := i
			for j < n && sql[j] != '\n' {
				j++
			}
			tokens = append(tokens, sqlToken{typ: tokenComment, text: sql[i:j]})
			i = j
			continue
		}

		// Block comment /* ... */
		if i+1 < n && sql[i] == '/' && sql[i+1] == '*' {
			j := i + 2
			for j+1 < n && !(sql[j] == '*' && sql[j+1] == '/') {
				j++
			}
			if j+1 < n {
				j += 2 // consume */
			}
			tokens = append(tokens, sqlToken{typ: tokenComment, text: sql[i:j]})
			i = j
			continue
		}

		// Single-quoted string '...'
		if ch == '\'' {
			j := i + 1
			for j < n {
				if sql[j] == '\'' {
					j++
					if j < n && sql[j] == '\'' { // escaped quote ''
						j++
						continue
					}
					break
				}
				j++
			}
			tokens = append(tokens, sqlToken{typ: tokenString, text: sql[i:j]})
			i = j
			continue
		}

		// Double-quoted identifier "..."
		if ch == '"' {
			j := i + 1
			for j < n && sql[j] != '"' {
				j++
			}
			if j < n {
				j++ // consume closing "
			}
			tokens = append(tokens, sqlToken{typ: tokenWord, text: sql[i:j]})
			i = j
			continue
		}

		// Back-quoted identifier `...`
		if ch == '`' {
			j := i + 1
			for j < n && sql[j] != '`' {
				j++
			}
			if j < n {
				j++ // consume closing `
			}
			tokens = append(tokens, sqlToken{typ: tokenWord, text: sql[i:j]})
			i = j
			continue
		}

		// Number
		if ch >= '0' && ch <= '9' {
			j := i
			for j < n && (sql[j] >= '0' && sql[j] <= '9' || sql[j] == '.' || sql[j] == 'e' || sql[j] == 'E') {
				j++
			}
			tokens = append(tokens, sqlToken{typ: tokenNumber, text: sql[i:j]})
			i = j
			continue
		}

		// Word (identifier or keyword)
		if isIdentStart(rune(ch)) {
			j := i
			for j < n && isIdentPart(rune(sql[j])) {
				j++
			}
			tokens = append(tokens, sqlToken{typ: tokenWord, text: sql[i:j]})
			i = j
			continue
		}

		// Everything else is an operator / punctuation (single char)
		tokens = append(tokens, sqlToken{typ: tokenOperator, text: string(ch)})
		i++
	}
	return tokens
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// formatSQL reformats tokenized SQL with indentation and keyword casing.
func formatSQL(tokens []sqlToken, indent string, uppercase bool) string {
	// Collect non-whitespace tokens for reformatting.
	var words []sqlToken
	for _, t := range tokens {
		if t.typ != tokenWhitespace {
			words = append(words, t)
		}
	}

	// Apply keyword casing.
	for i, w := range words {
		if w.typ == tokenWord {
			up := strings.ToUpper(w.text)
			if sqlKeywordSet[up] {
				if uppercase {
					words[i].text = up
				} else {
					words[i].text = strings.ToLower(w.text)
				}
			}
		}
	}

	// Build formatted output with newlines on major clause keywords.
	var sb strings.Builder
	depth := 0

	newlineKeywords := map[string]bool{
		"SELECT": true, "FROM": true, "WHERE": true, "ORDER": true, "GROUP": true,
		"HAVING": true, "LIMIT": true, "OFFSET": true, "UNION": true, "EXCEPT": true,
		"INTERSECT": true, "INSERT": true, "UPDATE": true, "DELETE": true,
		"SET": true, "VALUES": true, "RETURNING": true, "WITH": true,
	}

	joinKeywords := map[string]bool{
		"JOIN": true, "LEFT": true, "RIGHT": true, "INNER": true, "OUTER": true,
		"FULL": true, "CROSS": true, "NATURAL": true,
	}

	first := true
	for i, w := range words {
		up := strings.ToUpper(w.text)

		// Track parens
		if w.typ == tokenOperator && w.text == "(" {
			depth++
		}

		if !first && depth == 0 {
			if newlineKeywords[up] || joinKeywords[up] {
				sb.WriteString("\n")
				sb.WriteString(strings.Repeat(indent, 1))
			} else if w.text == "," {
				// comma stays inline
			} else {
				// add space between tokens if needed
				if i > 0 && !needsNoSpace(words[i-1], w) {
					sb.WriteString(" ")
				}
			}
		} else if !first {
			if !needsNoSpace(words[i-1], w) {
				sb.WriteString(" ")
			}
		}

		sb.WriteString(w.text)
		first = false

		if w.typ == tokenOperator && w.text == ")" {
			if depth > 0 {
				depth--
			}
		}
	}

	return sb.String()
}

// needsNoSpace returns true if no space should be placed between prev and curr.
func needsNoSpace(prev, curr sqlToken) bool {
	// No space before comma, semicolon, closing paren
	if curr.text == "," || curr.text == ";" || curr.text == ")" {
		return true
	}
	// No space after opening paren
	if prev.text == "(" {
		return true
	}
	// No space before opening paren after word (function call)
	if curr.text == "(" && prev.typ == tokenWord {
		return true
	}
	return false
}

// lintSQL produces warnings about common SQL issues.
func lintSQL(sql string, upperTokens []sqlToken) []string {
	var warnings []string
	upper := strings.ToUpper(sql)

	// Warn on SELECT *
	if matched, _ := regexp.MatchString(`(?i)SELECT\s+\*`, sql); matched {
		warnings = append(warnings, "SELECT * detected: consider selecting specific columns")
	}

	// Check if UPDATE/DELETE has WHERE clause
	if strings.Contains(upper, "UPDATE ") && !strings.Contains(upper, "WHERE") {
		warnings = append(warnings, "UPDATE without WHERE clause: all rows will be affected")
	}
	if strings.Contains(upper, "DELETE ") && !strings.Contains(upper, "WHERE") {
		warnings = append(warnings, "DELETE without WHERE clause: all rows will be deleted")
	}

	// Warn on cartesian joins (FROM with comma-separated tables without explicit join)
	_ = upperTokens
	fromRe := regexp.MustCompile(`(?i)FROM\s+\w+\s*,\s*\w+`)
	if fromRe.MatchString(sql) {
		warnings = append(warnings, "Potential cartesian join detected: use explicit JOIN syntax instead of comma-separated tables")
	}

	return warnings
}

// SQLFormat formats and lints a SQL statement.
func SQLFormat(_ context.Context, input SQLFormatInput) string {
	if strings.TrimSpace(input.SQL) == "" {
		return errResult("sql is required")
	}

	dialect := input.Dialect
	if dialect == "" {
		dialect = "generic"
	}
	validDialects := map[string]bool{"postgresql": true, "mysql": true, "sqlite": true, "generic": true}
	if !validDialects[dialect] {
		return errResult("dialect must be one of: postgresql, mysql, sqlite, generic")
	}

	indent := input.Indent
	if indent == "" {
		indent = "  "
	}

	uppercase := input.UppercaseKeyword
	// Default is true — the MCP layer should set it, but if it arrives as zero-value (false)
	// we check if the field was explicitly set. Since Go booleans default to false, we treat
	// uppercase_keywords omission as "true" only when the field is the zero value AND the
	// caller didn't explicitly send false. We follow the spec: default true.
	// (At the MCP layer we use mcp.ParseBoolean with default true.)

	tokens := tokenizeSQL(input.SQL)
	warnings := lintSQL(input.SQL, tokens)
	formatted := formatSQL(tokens, indent, uppercase)

	out := SQLFormatOutput{
		Result:   formatted,
		Warnings: warnings,
	}
	if out.Warnings == nil {
		out.Warnings = []string{}
	}
	return resultJSON(out)
}

// ─── backend_conn_string ─────────────────────────────────────────────────────

// ConnStringInput is the input schema for the backend_conn_string tool.
type ConnStringInput struct {
	Operation        string            `json:"operation"`         // build | parse
	DBType           string            `json:"db_type"`           // postgresql | mysql | mongodb | redis
	ConnectionString string            `json:"connection_string"` // for parse
	Host             string            `json:"host"`
	Port             int               `json:"port"`
	Database         string            `json:"database"`
	Username         string            `json:"username"`
	Password         string            `json:"password"`
	Options          map[string]string `json:"options"`
}

// ConnStringBuildOutput is the output for the build operation.
type ConnStringBuildOutput struct {
	ConnectionString string `json:"connection_string"`
}

// ConnStringParseOutput is the output for the parse operation.
type ConnStringParseOutput struct {
	Host     string            `json:"host"`
	Port     int               `json:"port"`
	Database string            `json:"database"`
	Username string            `json:"username"`
	Options  map[string]string `json:"options"`
}

// ConnString builds or parses a database connection string.
func ConnString(_ context.Context, input ConnStringInput) string {
	op := input.Operation
	if op == "" {
		op = "build"
	}
	if op != "build" && op != "parse" {
		return errResult("operation must be 'build' or 'parse'")
	}

	dbType := strings.ToLower(input.DBType)
	validTypes := map[string]bool{"postgresql": true, "mysql": true, "mongodb": true, "redis": true}
	if !validTypes[dbType] {
		return errResult("db_type must be one of: postgresql, mysql, mongodb, redis")
	}

	if op == "build" {
		return buildConnString(input, dbType)
	}
	return parseConnString(input, dbType)
}

func buildConnString(input ConnStringInput, dbType string) string {
	host := input.Host
	if host == "" {
		host = "localhost"
	}

	var dsn string

	switch dbType {
	case "postgresql":
		port := input.Port
		if port == 0 {
			port = 5432
		}
		userInfo := ""
		if input.Username != "" || input.Password != "" {
			userInfo = url.QueryEscape(input.Username) + ":" + url.QueryEscape(input.Password) + "@"
		}
		dsn = fmt.Sprintf("postgresql://%s%s:%d/%s", userInfo, host, port, input.Database)
		if len(input.Options) > 0 {
			params := url.Values{}
			for k, v := range input.Options {
				params.Set(k, v)
			}
			dsn += "?" + params.Encode()
		}

	case "mysql":
		port := input.Port
		if port == 0 {
			port = 3306
		}
		userInfo := ""
		if input.Username != "" {
			userInfo = input.Username
			if input.Password != "" {
				userInfo += ":" + input.Password
			}
			userInfo += "@"
		}
		dsn = fmt.Sprintf("%stcp(%s:%d)/%s", userInfo, host, port, input.Database)
		if len(input.Options) > 0 {
			params := url.Values{}
			for k, v := range input.Options {
				params.Set(k, v)
			}
			dsn += "?" + params.Encode()
		}

	case "mongodb":
		port := input.Port
		if port == 0 {
			port = 27017
		}
		userInfo := ""
		if input.Username != "" || input.Password != "" {
			userInfo = url.QueryEscape(input.Username) + ":" + url.QueryEscape(input.Password) + "@"
		}
		dsn = fmt.Sprintf("mongodb://%s%s:%d/%s", userInfo, host, port, input.Database)
		if len(input.Options) > 0 {
			params := url.Values{}
			for k, v := range input.Options {
				params.Set(k, v)
			}
			dsn += "?" + params.Encode()
		}

	case "redis":
		port := input.Port
		if port == 0 {
			port = 6379
		}
		db := "0"
		if input.Database != "" {
			db = input.Database
		}
		pw := ""
		if input.Password != "" {
			pw = ":" + url.QueryEscape(input.Password) + "@"
		}
		dsn = fmt.Sprintf("redis://%s%s:%d/%s", pw, host, port, db)
	}

	return resultJSON(ConnStringBuildOutput{ConnectionString: dsn})
}

func parseConnString(input ConnStringInput, dbType string) string {
	cs := strings.TrimSpace(input.ConnectionString)
	if cs == "" {
		return errResult("connection_string is required for parse operation")
	}

	out := ConnStringParseOutput{
		Options: map[string]string{},
	}

	switch dbType {
	case "postgresql", "mongodb":
		// Format: scheme://user:pass@host:port/dbname?opts
		u, err := url.Parse(cs)
		if err != nil {
			return errResult("failed to parse connection string: " + err.Error())
		}
		out.Host = u.Hostname()
		if p := u.Port(); p != "" {
			out.Port, _ = strconv.Atoi(p)
		}
		if u.User != nil {
			out.Username = u.User.Username()
		}
		out.Database = strings.TrimPrefix(u.Path, "/")
		for k, vals := range u.Query() {
			if len(vals) > 0 {
				out.Options[k] = vals[0]
			}
		}

	case "mysql":
		// Format: user:pass@tcp(host:port)/dbname?opts
		// Parse: extract userinfo before @tcp, then host:port inside (), then /dbname?opts
		at := strings.Index(cs, "@tcp(")
		if at < 0 {
			// Try simple URL parse as fallback
			u, err := url.Parse("mysql://" + cs)
			if err != nil {
				return errResult("failed to parse MySQL connection string: " + err.Error())
			}
			out.Host = u.Hostname()
			if p := u.Port(); p != "" {
				out.Port, _ = strconv.Atoi(p)
			}
			if u.User != nil {
				out.Username = u.User.Username()
			}
			out.Database = strings.TrimPrefix(u.Path, "/")
			break
		}
		userInfo := cs[:at]
		if colon := strings.Index(userInfo, ":"); colon >= 0 {
			out.Username = userInfo[:colon]
		} else {
			out.Username = userInfo
		}
		rest := cs[at+5:] // after "@tcp("
		closeParen := strings.Index(rest, ")")
		if closeParen < 0 {
			return errResult("invalid MySQL connection string: missing closing paren in host")
		}
		hostPort := rest[:closeParen]
		if colon := strings.LastIndex(hostPort, ":"); colon >= 0 {
			out.Host = hostPort[:colon]
			out.Port, _ = strconv.Atoi(hostPort[colon+1:])
		} else {
			out.Host = hostPort
		}
		dbAndOpts := rest[closeParen+1:]
		if strings.HasPrefix(dbAndOpts, "/") {
			dbAndOpts = dbAndOpts[1:]
		}
		if q := strings.Index(dbAndOpts, "?"); q >= 0 {
			out.Database = dbAndOpts[:q]
			vals, _ := url.ParseQuery(dbAndOpts[q+1:])
			for k, vs := range vals {
				if len(vs) > 0 {
					out.Options[k] = vs[0]
				}
			}
		} else {
			out.Database = dbAndOpts
		}

	case "redis":
		// Format: redis://:pass@host:port/db
		u, err := url.Parse(cs)
		if err != nil {
			return errResult("failed to parse Redis connection string: " + err.Error())
		}
		out.Host = u.Hostname()
		if p := u.Port(); p != "" {
			out.Port, _ = strconv.Atoi(p)
		}
		if u.User != nil {
			out.Username = u.User.Username()
		}
		out.Database = strings.TrimPrefix(u.Path, "/")
	}

	return resultJSON(out)
}

// ─── backend_log_parse ───────────────────────────────────────────────────────

// LogParseInput is the input schema for the backend_log_parse tool.
type LogParseInput struct {
	Log       string                 `json:"log"`
	Format    string                 `json:"format"`     // json | ndjson | apache | nginx | auto
	Filter    map[string]interface{} `json:"filter"`     // key/value filter
	StartTime string                 `json:"start_time"` // ISO8601
	EndTime   string                 `json:"end_time"`   // ISO8601
	Limit     int                    `json:"limit"`
}

// LogParseOutput is the output for the backend_log_parse tool.
type LogParseOutput struct {
	Entries        []map[string]interface{} `json:"entries"`
	Total          int                      `json:"total"`
	Matched        int                      `json:"matched"`
	FormatDetected string                   `json:"format_detected"`
}

// Apache Combined Log Format regex.
var apacheRe = regexp.MustCompile(
	`^(\S+)\s+\S+\s+(\S+)\s+\[([^\]]+)\]\s+"([^"]+)"\s+(\d+)\s+(\S+)(?:\s+"([^"]+)"\s+"([^"]+)")?`,
)

// Nginx error log regex (basic).
var nginxErrorRe = regexp.MustCompile(
	`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})\s+\[(\w+)\]\s+(.+)$`,
)

// detectLogFormat tries to determine the log format from the first non-empty line.
func detectLogFormat(log string) string {
	scanner := bufio.NewScanner(strings.NewReader(log))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "{") {
			return "ndjson"
		}
		if apacheRe.MatchString(line) {
			return "apache"
		}
		if nginxErrorRe.MatchString(line) {
			return "nginx"
		}
		return "text"
	}
	return "text"
}

// parseLogLine parses a single log line according to the given format.
func parseLogLine(line, format string) map[string]interface{} {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	switch format {
	case "json", "ndjson":
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return map[string]interface{}{"_raw": line, "_parse_error": err.Error()}
		}
		return entry

	case "apache":
		m := apacheRe.FindStringSubmatch(line)
		if m == nil {
			return map[string]interface{}{"_raw": line}
		}
		entry := map[string]interface{}{
			"remote_addr": m[1],
			"user":        m[2],
			"time_local":  m[3],
			"request":     m[4],
			"status":      m[5],
			"body_bytes":  m[6],
		}
		if len(m) > 7 && m[7] != "" {
			entry["http_referer"] = m[7]
		}
		if len(m) > 8 && m[8] != "" {
			entry["http_user_agent"] = m[8]
		}
		return entry

	case "nginx":
		m := nginxErrorRe.FindStringSubmatch(line)
		if m == nil {
			return map[string]interface{}{"_raw": line}
		}
		return map[string]interface{}{
			"time":    m[1],
			"level":   m[2],
			"message": m[3],
		}

	default:
		return map[string]interface{}{"_raw": line}
	}
}

// extractTime attempts to extract a time from a log entry.
// Common field names: time, timestamp, @timestamp, time_local, datetime.
func extractTime(entry map[string]interface{}) (time.Time, bool) {
	timeFields := []string{"time", "timestamp", "@timestamp", "time_local", "datetime", "date"}
	for _, f := range timeFields {
		if v, ok := entry[f]; ok {
			if s, ok := v.(string); ok {
				layouts := []string{
					time.RFC3339,
					time.RFC3339Nano,
					"2006/01/02 15:04:05",
					"02/Jan/2006:15:04:05 -0700",
					"2006-01-02T15:04:05",
					"2006-01-02 15:04:05",
				}
				for _, layout := range layouts {
					if t, err := time.Parse(layout, s); err == nil {
						return t, true
					}
				}
			}
		}
	}
	return time.Time{}, false
}

// matchesFilter checks if an entry passes all filter conditions.
func matchesFilter(entry map[string]interface{}, filter map[string]interface{}) bool {
	for k, v := range filter {
		entryVal, ok := entry[k]
		if !ok {
			return false
		}
		ev, _ := json.Marshal(entryVal)
		fv, _ := json.Marshal(v)
		if string(ev) != string(fv) {
			// Try string comparison
			es := fmt.Sprintf("%v", entryVal)
			fs := fmt.Sprintf("%v", v)
			if es != fs {
				return false
			}
		}
	}
	return true
}

// LogParse parses multiline log content and returns filtered entries.
func LogParse(_ context.Context, input LogParseInput) string {
	if strings.TrimSpace(input.Log) == "" {
		return errResult("log is required")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}

	format := input.Format
	if format == "" || format == "auto" {
		format = detectLogFormat(input.Log)
	}

	var startTime, endTime time.Time
	var hasStart, hasEnd bool
	if input.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, input.StartTime); err == nil {
			startTime = t
			hasStart = true
		} else if t, err := time.Parse("2006-01-02T15:04:05", input.StartTime); err == nil {
			startTime = t
			hasStart = true
		}
	}
	if input.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, input.EndTime); err == nil {
			endTime = t
			hasEnd = true
		} else if t, err := time.Parse("2006-01-02T15:04:05", input.EndTime); err == nil {
			endTime = t
			hasEnd = true
		}
	}

	var allEntries []map[string]interface{}
	scanner := bufio.NewScanner(strings.NewReader(input.Log))
	for scanner.Scan() {
		line := scanner.Text()
		entry := parseLogLine(line, format)
		if entry == nil {
			continue
		}
		allEntries = append(allEntries, entry)
	}

	total := len(allEntries)
	var matched []map[string]interface{}

	for _, entry := range allEntries {
		// Time range filter
		if hasStart || hasEnd {
			if t, ok := extractTime(entry); ok {
				if hasStart && t.Before(startTime) {
					continue
				}
				if hasEnd && t.After(endTime) {
					continue
				}
			}
		}

		// Field filter
		if len(input.Filter) > 0 && !matchesFilter(entry, input.Filter) {
			continue
		}

		matched = append(matched, entry)
		if len(matched) >= limit {
			break
		}
	}

	if matched == nil {
		matched = []map[string]interface{}{}
	}

	out := LogParseOutput{
		Entries:        matched,
		Total:          total,
		Matched:        len(matched),
		FormatDetected: format,
	}
	return resultJSON(out)
}

// ─── backend_env_inspect ─────────────────────────────────────────────────────

// EnvKeySchema defines the expected schema for an env key.
type EnvKeySchema struct {
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Pattern     string `json:"pattern"`
}

// EnvInspectInput is the input schema for the backend_env_inspect tool.
type EnvInspectInput struct {
	EnvContent string `json:"env_content"`
	Schema     string `json:"schema"`    // JSON object of key -> EnvKeySchema
	Operation  string `json:"operation"` // validate | generate_example
}

// EnvValidateOutput is the output for the validate operation.
type EnvValidateOutput struct {
	Valid           bool             `json:"valid"`
	MissingRequired []string         `json:"missing_required"`
	UnknownKeys     []string         `json:"unknown_keys"`
	InvalidFormat   []EnvFormatError `json:"invalid_format"`
}

// EnvFormatError describes a key that failed pattern validation.
type EnvFormatError struct {
	Key   string `json:"key"`
	Error string `json:"error"`
}

// EnvGenerateOutput is the output for the generate_example operation.
type EnvGenerateOutput struct {
	Example string `json:"example"`
}

// parseEnvContent parses a .env file content into a map of key->value.
// Supports: KEY=VALUE, quoted values, comments (#), and multiline with \.
func parseEnvContent(content string) map[string]string {
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(content))
	var continuation string
	var continuationKey string

	for scanner.Scan() {
		line := scanner.Text()

		// Handle continuation from previous line
		if continuation != "" {
			if strings.HasSuffix(line, "\\") {
				continuation += strings.TrimSuffix(line, "\\")
				continue
			}
			result[continuationKey] = continuation + line
			continuation = ""
			continuationKey = ""
			continue
		}

		// Skip empty lines and comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Find first = sign
		eq := strings.Index(trimmed, "=")
		if eq < 0 {
			continue
		}

		key := strings.TrimSpace(trimmed[:eq])
		val := strings.TrimSpace(trimmed[eq+1:])

		// Handle multiline continuation
		if strings.HasSuffix(val, "\\") {
			continuation = strings.TrimSuffix(val, "\\")
			continuationKey = key
			continue
		}

		// Handle quoted values
		val = unquoteEnvValue(val)

		result[key] = val
	}

	return result
}

// unquoteEnvValue removes surrounding quotes from an env value.
func unquoteEnvValue(val string) string {
	if len(val) >= 2 {
		if (val[0] == '"' && val[len(val)-1] == '"') ||
			(val[0] == '\'' && val[len(val)-1] == '\'') {
			return val[1 : len(val)-1]
		}
	}
	return val
}

// EnvInspect validates or generates an example .env file.
func EnvInspect(_ context.Context, input EnvInspectInput) string {
	if strings.TrimSpace(input.EnvContent) == "" {
		return errResult("env_content is required")
	}

	op := input.Operation
	if op == "" {
		op = "validate"
	}
	if op != "validate" && op != "generate_example" {
		return errResult("operation must be 'validate' or 'generate_example'")
	}

	// Parse schema if provided
	var schema map[string]EnvKeySchema
	if strings.TrimSpace(input.Schema) != "" {
		if err := json.Unmarshal([]byte(input.Schema), &schema); err != nil {
			return errResult("invalid schema JSON: " + err.Error())
		}
	}

	parsed := parseEnvContent(input.EnvContent)

	if op == "generate_example" {
		return generateEnvExample(parsed, schema)
	}

	return validateEnv(parsed, schema)
}

func validateEnv(parsed map[string]string, schema map[string]EnvKeySchema) string {
	out := EnvValidateOutput{
		MissingRequired: []string{},
		UnknownKeys:     []string{},
		InvalidFormat:   []EnvFormatError{},
	}

	if schema != nil {
		// Check required keys and pattern validation
		for key, spec := range schema {
			val, exists := parsed[key]
			if !exists {
				if spec.Required {
					out.MissingRequired = append(out.MissingRequired, key)
				}
				continue
			}
			// Pattern validation
			if spec.Pattern != "" {
				re, err := regexp.Compile(spec.Pattern)
				if err != nil {
					out.InvalidFormat = append(out.InvalidFormat, EnvFormatError{
						Key:   key,
						Error: "invalid pattern: " + err.Error(),
					})
					continue
				}
				if !re.MatchString(val) {
					out.InvalidFormat = append(out.InvalidFormat, EnvFormatError{
						Key:   key,
						Error: fmt.Sprintf("value does not match pattern %q", spec.Pattern),
					})
				}
			}
		}

		// Check for unknown keys
		for key := range parsed {
			if _, defined := schema[key]; !defined {
				out.UnknownKeys = append(out.UnknownKeys, key)
			}
		}
	}

	out.Valid = len(out.MissingRequired) == 0 && len(out.InvalidFormat) == 0
	return resultJSON(out)
}

func generateEnvExample(parsed map[string]string, schema map[string]EnvKeySchema) string {
	var sb strings.Builder
	sb.WriteString("# Generated .env.example\n\n")

	// Use schema keys first (in order of schema definition) if schema provided
	written := make(map[string]bool)

	if schema != nil {
		for key, spec := range schema {
			if spec.Description != "" {
				sb.WriteString("# " + spec.Description)
				if spec.Required {
					sb.WriteString(" (required)")
				}
				sb.WriteString("\n")
			}
			placeholder := ""
			if spec.Pattern != "" {
				placeholder = "# pattern: " + spec.Pattern
			}
			if placeholder != "" {
				sb.WriteString(placeholder + "\n")
			}
			sb.WriteString(key + "=\n\n")
			written[key] = true
		}
	}

	// Write remaining keys from parsed content
	for key := range parsed {
		if !written[key] {
			sb.WriteString(key + "=\n")
		}
	}

	return resultJSON(EnvGenerateOutput{Example: sb.String()})
}

// ─── backend_mq_payload ──────────────────────────────────────────────────────

// MQPayloadInput is the input schema for the backend_mq_payload tool.
type MQPayloadInput struct {
	Broker    string                 `json:"broker"`    // kafka | rabbitmq | sqs
	Operation string                 `json:"operation"` // build | serialize | format
	Topic     string                 `json:"topic"`
	Payload   string                 `json:"payload"` // JSON body
	Headers   map[string]string      `json:"headers"`
	Options   map[string]interface{} `json:"options"` // broker-specific
}

// MQPayloadOutput is the output for the backend_mq_payload tool.
type MQPayloadOutput struct {
	Envelope   interface{} `json:"envelope"`
	Serialized string      `json:"serialized"`
	Broker     string      `json:"broker"`
}

// MQPayload builds a message queue payload envelope for the specified broker.
func MQPayload(_ context.Context, input MQPayloadInput) string {
	broker := strings.ToLower(input.Broker)
	validBrokers := map[string]bool{"kafka": true, "rabbitmq": true, "sqs": true}
	if !validBrokers[broker] {
		return errResult("broker must be one of: kafka, rabbitmq, sqs")
	}

	op := input.Operation
	if op == "" {
		op = "build"
	}
	validOps := map[string]bool{"build": true, "serialize": true, "format": true}
	if !validOps[op] {
		return errResult("operation must be one of: build, serialize, format")
	}

	// Parse payload as JSON if provided
	var payloadBody interface{}
	if strings.TrimSpace(input.Payload) != "" {
		if err := json.Unmarshal([]byte(input.Payload), &payloadBody); err != nil {
			// treat as raw string
			payloadBody = input.Payload
		}
	}

	headers := input.Headers
	if headers == nil {
		headers = map[string]string{}
	}

	opts := input.Options
	if opts == nil {
		opts = map[string]interface{}{}
	}

	var envelope interface{}

	switch broker {
	case "kafka":
		envelope = buildKafkaEnvelope(input.Topic, payloadBody, headers, opts)
	case "rabbitmq":
		envelope = buildRabbitMQEnvelope(input.Topic, payloadBody, headers, opts)
	case "sqs":
		envelope = buildSQSEnvelope(input.Topic, payloadBody, headers, opts)
	}

	serializedBytes, err := json.Marshal(envelope)
	if err != nil {
		return errResult("failed to serialize envelope: " + err.Error())
	}

	out := MQPayloadOutput{
		Envelope:   envelope,
		Serialized: string(serializedBytes),
		Broker:     broker,
	}
	return resultJSON(out)
}

// strOption extracts a string option from the options map.
func strOption(opts map[string]interface{}, key, fallback string) string {
	if v, ok := opts[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return fallback
}

// intOption extracts an int option from the options map.
func intOption(opts map[string]interface{}, key string, fallback int) int {
	if v, ok := opts[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case int64:
			return int(n)
		}
	}
	return fallback
}

// buildKafkaEnvelope constructs a Kafka message envelope.
func buildKafkaEnvelope(topic string, payload interface{}, headers map[string]string, opts map[string]interface{}) map[string]interface{} {
	key := strOption(opts, "key", "")
	partition := intOption(opts, "partition", 0)

	envelope := map[string]interface{}{
		"topic":     topic,
		"key":       key,
		"value":     payload,
		"headers":   headers,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"partition": partition,
	}
	return envelope
}

// buildRabbitMQEnvelope constructs a RabbitMQ message envelope.
func buildRabbitMQEnvelope(routingKey string, payload interface{}, headers map[string]string, opts map[string]interface{}) map[string]interface{} {
	exchange := strOption(opts, "exchange", "")
	contentType := strOption(opts, "content_type", "application/json")
	deliveryMode := intOption(opts, "delivery_mode", 2)
	correlationID := strOption(opts, "correlation_id", "")
	replyTo := strOption(opts, "reply_to", "")
	expiration := strOption(opts, "expiration", "")
	messageID := strOption(opts, "message_id", "")
	priority := intOption(opts, "priority", 0)

	properties := map[string]interface{}{
		"content_type":  contentType,
		"delivery_mode": deliveryMode,
		"headers":       headers,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	}
	if correlationID != "" {
		properties["correlation_id"] = correlationID
	}
	if replyTo != "" {
		properties["reply_to"] = replyTo
	}
	if expiration != "" {
		properties["expiration"] = expiration
	}
	if messageID != "" {
		properties["message_id"] = messageID
	}
	if priority > 0 {
		properties["priority"] = priority
	}

	return map[string]interface{}{
		"exchange":    exchange,
		"routing_key": routingKey,
		"properties":  properties,
		"body":        payload,
	}
}

// buildSQSEnvelope constructs an SQS message envelope.
func buildSQSEnvelope(queueName string, payload interface{}, headers map[string]string, opts map[string]interface{}) map[string]interface{} {
	queueURL := strOption(opts, "queue_url", queueName)
	messageGroupID := strOption(opts, "message_group_id", "")
	messageDeduplicationID := strOption(opts, "message_deduplication_id", "")

	// Serialize payload as string for SQS MessageBody
	bodyStr := ""
	if payload != nil {
		b, _ := json.Marshal(payload)
		bodyStr = string(b)
	}

	// Build MessageAttributes from headers
	msgAttrs := map[string]interface{}{}
	for k, v := range headers {
		msgAttrs[k] = map[string]interface{}{
			"DataType":    "String",
			"StringValue": v,
		}
	}

	envelope := map[string]interface{}{
		"QueueUrl":          queueURL,
		"MessageBody":       bodyStr,
		"MessageAttributes": msgAttrs,
	}
	if messageGroupID != "" {
		envelope["MessageGroupId"] = messageGroupID
	}
	if messageDeduplicationID != "" {
		envelope["MessageDeduplicationId"] = messageDeduplicationID
	}

	return envelope
}

// ─── backend_cidr_subnet ─────────────────────────────────────────────────────

// CIDRSubnetInput is the input schema for the backend_cidr_subnet tool.
type CIDRSubnetInput struct {
	CIDR       string `json:"cidr"`
	IncludeAll bool   `json:"include_all"`
	Limit      int    `json:"limit"`
}

// CIDRSubnetOutput is the output schema for the backend_cidr_subnet tool.
type CIDRSubnetOutput struct {
	CIDR        string   `json:"cidr"`
	Network     string   `json:"network"`
	Broadcast   string   `json:"broadcast"`
	Netmask     string   `json:"netmask"`
	Prefix      int      `json:"prefix"`
	TotalIPs    uint64   `json:"total_ips"`
	UsableIPs   uint64   `json:"usable_ips"`
	FirstUsable string   `json:"first_usable"`
	LastUsable  string   `json:"last_usable"`
	AvailableIP []string `json:"available_ips,omitempty"`
	Truncated   bool     `json:"truncated,omitempty"`
}

// CIDRSubnet calculates network details and optionally lists usable host IPs.
func CIDRSubnet(_ context.Context, input CIDRSubnetInput) string {
	cidr := strings.TrimSpace(input.CIDR)
	if cidr == "" {
		return errResult("cidr is required")
	}

	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return errResult("invalid cidr: " + err.Error())
	}

	ipv4 := ip.To4()
	if ipv4 == nil {
		return errResult("only IPv4 CIDR blocks are supported")
	}

	ones, bits := ipNet.Mask.Size()
	if bits != 32 {
		return errResult("only IPv4 CIDR blocks are supported")
	}

	network := ipv4.Mask(ipNet.Mask).To4()
	broadcast := make(net.IP, len(network))
	copy(broadcast, network)
	for i := 0; i < 4; i++ {
		broadcast[i] = network[i] | ^ipNet.Mask[i]
	}

	networkU := ipToUint32(network)
	broadcastU := ipToUint32(broadcast)
	totalIPs := uint64(broadcastU-networkU) + 1

	usableIPs := totalIPs
	firstU := networkU
	lastU := broadcastU
	if ones <= 30 {
		usableIPs = totalIPs - 2
		firstU = networkU + 1
		lastU = broadcastU - 1
	}

	out := CIDRSubnetOutput{
		CIDR:        cidr,
		Network:     uint32ToIPv4(networkU).String(),
		Broadcast:   uint32ToIPv4(broadcastU).String(),
		Netmask:     net.IP(ipNet.Mask).String(),
		Prefix:      ones,
		TotalIPs:    totalIPs,
		UsableIPs:   usableIPs,
		FirstUsable: uint32ToIPv4(firstU).String(),
		LastUsable:  uint32ToIPv4(lastU).String(),
	}

	if input.IncludeAll {
		limit := input.Limit
		if limit <= 0 {
			limit = 256
		}

		available := make([]string, 0)
		for host := firstU; host <= lastU; host++ {
			if len(available) >= limit {
				out.Truncated = true
				break
			}
			available = append(available, uint32ToIPv4(host).String())
			if host == lastU {
				break
			}
		}
		out.AvailableIP = available
	}

	return resultJSON(out)
}

func ipToUint32(ip net.IP) uint32 {
	return binary.BigEndian.Uint32(ip.To4())
}

func uint32ToIPv4(v uint32) net.IP {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return net.IPv4(b[0], b[1], b[2], b[3]).To4()
}
