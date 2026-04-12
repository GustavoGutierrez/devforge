# Backend Utility Tools

MCP tools for backend and infrastructure workflows: SQL formatting, connection string management, log parsing, environment inspection, message queue payload building, and CIDR subnet calculations.

---

## `backend_sql_format`

Format and lint a SQL statement with configurable indentation, keyword casing, and dialect-aware warnings.

### Input

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `sql` | string | ✅ | — | SQL statement to format |
| `dialect` | string | | `generic` | `postgresql \| mysql \| sqlite \| generic` |
| `indent` | string | | `"  "` | Indentation string (two spaces) |
| `uppercase_keywords` | bool | | `true` | Uppercase SQL keywords |

### Output

```json
{
  "result": "SELECT id, name\n  FROM users\n  WHERE id = 1",
  "warnings": ["SELECT * detected: consider selecting specific columns"]
}
```

### Lint Warnings

- **SELECT \***: Warns to select specific columns instead
- **UPDATE without WHERE**: Warns that all rows will be affected
- **DELETE without WHERE**: Warns that all rows will be deleted
- **Cartesian join**: Warns when comma-separated tables appear in FROM without an explicit JOIN

### Example

```json
{
  "sql": "select * from users where id = 1",
  "dialect": "postgresql",
  "uppercase_keywords": true
}
```

---

## `backend_conn_string`

Build or parse a database connection string (DSN) for PostgreSQL, MySQL, MongoDB, or Redis.

### Input

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `operation` | string | ✅ | — | `build \| parse` |
| `db_type` | string | ✅ | — | `postgresql \| mysql \| mongodb \| redis` |
| `connection_string` | string | (parse) | — | Connection string to parse |
| `host` | string | | `localhost` | Database host |
| `port` | int | | DB default | Database port |
| `database` | string | | `""` | Database name |
| `username` | string | | `""` | Username |
| `password` | string | | `""` | Password |
| `options` | object | | `{}` | Extra key/value connection parameters |

### Output (build)

```json
{
  "connection_string": "postgresql://admin:secret@localhost:5432/mydb?sslmode=require"
}
```

### Output (parse)

```json
{
  "host": "localhost",
  "port": 5432,
  "database": "mydb",
  "username": "admin",
  "options": { "sslmode": "require" }
}
```

### Connection String Formats

| DB Type | Format |
|---------|--------|
| PostgreSQL | `postgresql://user:pass@host:port/dbname?key=val` |
| MySQL | `user:pass@tcp(host:port)/dbname?key=val` |
| MongoDB | `mongodb://user:pass@host:port/dbname` |
| Redis | `redis://:password@host:port/db` |

### Default Ports

| DB Type | Default Port |
|---------|-------------|
| PostgreSQL | 5432 |
| MySQL | 3306 |
| MongoDB | 27017 |
| Redis | 6379 |

---

## `backend_log_parse`

Parse multiline log content in JSON, NDJSON, Apache Combined, or Nginx error format. Filter entries by field values and time range.

### Input

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `log` | string | ✅ | — | Multiline log content |
| `format` | string | | `auto` | `json \| ndjson \| apache \| nginx \| auto` |
| `filter` | object | | `{}` | Key/value pairs to filter entries by |
| `start_time` | string | | — | ISO8601 start time |
| `end_time` | string | | — | ISO8601 end time |
| `limit` | int | | `100` | Maximum entries to return |

### Output

```json
{
  "entries": [
    { "level": "error", "msg": "connection failed", "time": "2024-01-15T10:01:00Z" }
  ],
  "total": 3,
  "matched": 1,
  "format_detected": "ndjson"
}
```

### Auto-detection

When `format` is `auto`, the first non-empty line determines the format:
- Starts with `{` → `ndjson`
- Matches Apache combined log pattern → `apache`
- Matches Nginx error log pattern → `nginx`
- Otherwise → `text` (returns raw lines)

### Time Extraction

For time-range filtering, the tool looks for these fields: `time`, `timestamp`, `@timestamp`, `time_local`, `datetime`, `date`.

---

## `backend_env_inspect`

Validate a `.env` file against a schema, or generate a `.env.example` template from it.

### Input

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `env_content` | string | ✅ | — | Contents of the `.env` file |
| `schema` | string | | — | JSON object mapping keys to `{required, description, pattern}` |
| `operation` | string | | `validate` | `validate \| generate_example` |

### Schema Format

```json
{
  "DB_HOST": { "required": true, "description": "Database host" },
  "DB_PORT": { "required": true, "description": "Port number", "pattern": "^[0-9]+$" },
  "API_KEY": { "required": true, "description": "API authentication key" },
  "DEBUG":   { "required": false, "description": "Enable debug mode" }
}
```

### Output (validate)

```json
{
  "valid": false,
  "missing_required": ["API_KEY"],
  "unknown_keys": ["EXTRA_VAR"],
  "invalid_format": [
    { "key": "DB_PORT", "error": "value does not match pattern \"^[0-9]+$\"" }
  ]
}
```

### Output (generate_example)

```json
{
  "example": "# Generated .env.example\n\n# Database host (required)\nDB_HOST=\n\n..."
}
```

### .env Parsing Rules

- Lines starting with `#` are treated as comments
- Empty lines are skipped
- Values may be single-quoted (`'val'`) or double-quoted (`"val"`) — quotes are stripped
- Multiline values via trailing `\` are supported

---

## `backend_mq_payload`

Build a message queue payload envelope for Kafka, RabbitMQ, or SQS. Pure serialization — no actual broker connections are made.

### Input

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `broker` | string | ✅ | — | `kafka \| rabbitmq \| sqs` |
| `operation` | string | | `build` | `build \| serialize \| format` |
| `topic` | string | | `""` | Kafka topic / RabbitMQ routing key / SQS queue name or URL |
| `payload` | string | | — | JSON body of the message |
| `headers` | object | | `{}` | Message headers as key/value pairs |
| `options` | object | | `{}` | Broker-specific options (see below) |

### Output

```json
{
  "envelope": { ... },
  "serialized": "{\"topic\":\"user-events\",...}",
  "broker": "kafka"
}
```

### Broker-Specific Options

#### Kafka

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `key` | string | `""` | Message key for partitioning |
| `partition` | int | `0` | Target partition |

#### RabbitMQ

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `exchange` | string | `""` | Exchange name |
| `content_type` | string | `"application/json"` | Content-Type property |
| `delivery_mode` | int | `2` | `1` = transient, `2` = persistent |
| `correlation_id` | string | — | Correlation ID |
| `reply_to` | string | — | Reply-to queue |
| `expiration` | string | — | Message TTL |
| `message_id` | string | — | Message identifier |
| `priority` | int | — | Message priority |

#### SQS

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `queue_url` | string | (topic) | Full SQS queue URL |
| `message_group_id` | string | — | For FIFO queues |
| `message_deduplication_id` | string | — | For FIFO deduplication |

### Envelope Examples

**Kafka:**
```json
{
  "topic": "user-events",
  "key": "user-42",
  "value": { "event": "user.created", "user_id": 42 },
  "headers": { "content-type": "application/json" },
  "timestamp": "2024-01-15T10:00:00Z",
  "partition": 0
}
```

**RabbitMQ:**
```json
{
  "exchange": "events",
  "routing_key": "user.created",
  "properties": {
    "content_type": "application/json",
    "delivery_mode": 2,
    "headers": {},
    "timestamp": "2024-01-15T10:00:00Z"
  },
  "body": { "event": "user.created" }
}
```

**SQS:**
```json
{
  "QueueUrl": "https://sqs.us-east-1.amazonaws.com/123/my-queue",
  "MessageBody": "{\"event\":\"user.created\"}",
  "MessageAttributes": {
    "source": { "DataType": "String", "StringValue": "backend" }
  },
  "MessageGroupId": "group-1"
}
```

---

## `backend_cidr_subnet`

Calculate IPv4 CIDR subnet details and optionally list usable host IP addresses.

### Input

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `cidr` | string | ✅ | — | IPv4 CIDR block (for example `10.0.0.0/24`) |
| `include_all` | bool | | `true` | Include usable host list in response |
| `limit` | int | | `256` | Maximum hosts to return when `include_all=true` |

### Output

```json
{
  "cidr": "10.0.0.0/24",
  "network": "10.0.0.0",
  "broadcast": "10.0.0.255",
  "netmask": "255.255.255.0",
  "prefix": 24,
  "total_ips": 256,
  "usable_ips": 254,
  "first_usable": "10.0.0.1",
  "last_usable": "10.0.0.254",
  "available_ips": ["10.0.0.1", "10.0.0.2", "..."],
  "truncated": true
}
```

### Notes

- Only IPv4 is supported.
- `/31` and `/32` are handled as special cases for usable host calculation.
- Host listing respects `limit` to keep responses bounded.
