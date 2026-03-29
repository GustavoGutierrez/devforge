// Package main — registration of Security & Cryptography MCP tools.
package main

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"dev-forge-mcp/internal/tools/cryptoutil"
)

// registerCryptoUtilTools registers all cryptoutil MCP tools on the server.
func registerCryptoUtilTools(s *mcpserver.MCPServer, _ *mcpApp) {
	// ── crypto_hash ──────────────────────────────────────────────
	s.AddTool(mcp.NewTool("crypto_hash",
		mcp.WithDescription("Hash a string using SHA-256, SHA-512, MD5, or SHA-1 with hex or base64 output."),
		mcp.WithString("input", mcp.Required(), mcp.Description("String to hash")),
		mcp.WithString("algorithm", mcp.Description("Hash algorithm: sha256 | sha512 | md5 | sha1 (default sha256)")),
		mcp.WithString("encoding", mcp.Description("Output encoding: hex | base64 (default hex)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		in := cryptoutil.HashInput{
			Input:     mcp.ParseString(req, "input", ""),
			Algorithm: mcp.ParseString(req, "algorithm", "sha256"),
			Encoding:  mcp.ParseString(req, "encoding", "hex"),
		}
		return mcp.NewToolResultText(cryptoutil.Hash(ctx, in)), nil
	})

	// ── crypto_hmac ──────────────────────────────────────────────
	s.AddTool(mcp.NewTool("crypto_hmac",
		mcp.WithDescription("Compute an HMAC for a message using SHA-256 or SHA-512."),
		mcp.WithString("message", mcp.Required(), mcp.Description("Message to authenticate")),
		mcp.WithString("key", mcp.Required(), mcp.Description("HMAC secret key")),
		mcp.WithString("algorithm", mcp.Description("Algorithm: sha256 | sha512 (default sha256)")),
		mcp.WithString("encoding", mcp.Description("Output encoding: hex | base64 (default hex)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		in := cryptoutil.HMACInput{
			Message:   mcp.ParseString(req, "message", ""),
			Key:       mcp.ParseString(req, "key", ""),
			Algorithm: mcp.ParseString(req, "algorithm", "sha256"),
			Encoding:  mcp.ParseString(req, "encoding", "hex"),
		}
		return mcp.NewToolResultText(cryptoutil.HMAC(ctx, in)), nil
	})

	// ── crypto_jwt ───────────────────────────────────────────────
	s.AddTool(mcp.NewTool("crypto_jwt",
		mcp.WithDescription("Generate, decode, or verify JWT tokens using HS256 or HS512."),
		mcp.WithString("operation", mcp.Required(), mcp.Description("Operation: decode | verify | generate")),
		mcp.WithString("token", mcp.Description("JWT token string (required for decode/verify)")),
		mcp.WithString("secret", mcp.Description("HMAC secret (required for verify/generate)")),
		mcp.WithString("payload", mcp.Description("JSON payload object string (for generate)")),
		mcp.WithNumber("expiry_seconds", mcp.Description("Token expiry in seconds (default 3600, for generate)")),
		mcp.WithString("algorithm", mcp.Description("Algorithm: HS256 | HS512 (default HS256)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		expiry := 3600
		if e, ok := args["expiry_seconds"].(float64); ok {
			expiry = int(e)
		}
		in := cryptoutil.JWTInput{
			Operation:     mcp.ParseString(req, "operation", ""),
			Token:         mcp.ParseString(req, "token", ""),
			Secret:        mcp.ParseString(req, "secret", ""),
			Payload:       mcp.ParseString(req, "payload", ""),
			ExpirySeconds: expiry,
			Algorithm:     mcp.ParseString(req, "algorithm", "HS256"),
		}
		return mcp.NewToolResultText(cryptoutil.JWT(ctx, in)), nil
	})

	// ── crypto_password ──────────────────────────────────────────
	s.AddTool(mcp.NewTool("crypto_password",
		mcp.WithDescription("Hash or verify a password using bcrypt or Argon2id."),
		mcp.WithString("password", mcp.Required(), mcp.Description("Password to hash or verify")),
		mcp.WithString("operation", mcp.Required(), mcp.Description("Operation: hash | verify")),
		mcp.WithString("algorithm", mcp.Description("Algorithm: bcrypt | argon2id (default bcrypt)")),
		mcp.WithString("hash", mcp.Description("Existing hash (required for verify)")),
		mcp.WithNumber("cost", mcp.Description("bcrypt cost factor (default 12)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		cost := 12
		if c, ok := args["cost"].(float64); ok {
			cost = int(c)
		}
		in := cryptoutil.PasswordInput{
			Password:  mcp.ParseString(req, "password", ""),
			Operation: mcp.ParseString(req, "operation", ""),
			Algorithm: mcp.ParseString(req, "algorithm", "bcrypt"),
			Hash:      mcp.ParseString(req, "hash", ""),
			Cost:      cost,
		}
		return mcp.NewToolResultText(cryptoutil.Password(ctx, in)), nil
	})

	// ── crypto_keygen ────────────────────────────────────────────
	s.AddTool(mcp.NewTool("crypto_keygen",
		mcp.WithDescription("Generate RSA, EC, or Ed25519 key pairs in PEM or JWK format."),
		mcp.WithString("key_type", mcp.Required(), mcp.Description("Key type: rsa | ec | ed25519")),
		mcp.WithNumber("bits", mcp.Description("RSA key size: 2048 | 4096 (default 2048)")),
		mcp.WithString("curve", mcp.Description("EC curve: P-256 | P-384 (default P-256)")),
		mcp.WithString("format", mcp.Description("Output format: pem | jwk (default pem)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		bits := 2048
		if b, ok := args["bits"].(float64); ok {
			bits = int(b)
		}
		in := cryptoutil.KeygenInput{
			KeyType: mcp.ParseString(req, "key_type", ""),
			Bits:    bits,
			Curve:   mcp.ParseString(req, "curve", "P-256"),
			Format:  mcp.ParseString(req, "format", "pem"),
		}
		return mcp.NewToolResultText(cryptoutil.Keygen(ctx, in)), nil
	})

	// ── crypto_random ────────────────────────────────────────────
	s.AddTool(mcp.NewTool("crypto_random",
		mcp.WithDescription("Generate cryptographically secure random tokens, bytes, or numeric OTP codes."),
		mcp.WithString("kind", mcp.Required(), mcp.Description("Kind: token | bytes | otp")),
		mcp.WithNumber("length", mcp.Description("Length in bytes/digits (default 32 for token/bytes, 6 for otp)")),
		mcp.WithString("encoding", mcp.Description("Output encoding: hex | base64 | base64url (default hex, ignored for otp)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		length := 0
		if l, ok := args["length"].(float64); ok {
			length = int(l)
		}
		in := cryptoutil.RandomInput{
			Kind:     mcp.ParseString(req, "kind", ""),
			Length:   length,
			Encoding: mcp.ParseString(req, "encoding", "hex"),
		}
		return mcp.NewToolResultText(cryptoutil.Random(ctx, in)), nil
	})

	// ── crypto_mask ──────────────────────────────────────────────
	s.AddTool(mcp.NewTool("crypto_mask",
		mcp.WithDescription("Redact secrets (API keys, passwords, emails, credit cards, JWTs) from text using regex patterns."),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text to scan and redact")),
		mcp.WithArray("patterns", mcp.Description("Patterns to apply: api_key | password | email | credit_card | jwt | all (default [\"all\"])"), mcp.WithStringItems()),
		mcp.WithString("replacement", mcp.Description("Replacement string (default [REDACTED])")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		in := cryptoutil.MaskInput{
			Text:        mcp.ParseString(req, "text", ""),
			Replacement: mcp.ParseString(req, "replacement", "[REDACTED]"),
		}
		if patternsRaw, ok := args["patterns"]; ok {
			data, _ := json.Marshal(patternsRaw)
			json.Unmarshal(data, &in.Patterns)
		}
		return mcp.NewToolResultText(cryptoutil.Mask(ctx, in)), nil
	})
}
