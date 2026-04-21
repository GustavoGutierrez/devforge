package cryptoutil_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools/cryptoutil"
)

// ── crypto_hash ──────────────────────────────────────────────────────────────

func TestHash_SHA256_DefaultHex(t *testing.T) {
	in := cryptoutil.HashInput{Input: "hello", Algorithm: "sha256", Encoding: "hex"}
	result := cryptoutil.Hash(context.Background(), in)

	var out cryptoutil.HashOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	// Known SHA-256("hello") in hex
	const expected = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if out.Hash != expected {
		t.Errorf("wrong hash: got %q, want %q", out.Hash, expected)
	}
	if out.Algorithm != "sha256" {
		t.Errorf("wrong algorithm: %q", out.Algorithm)
	}
	if out.Encoding != "hex" {
		t.Errorf("wrong encoding: %q", out.Encoding)
	}
}

func TestHash_MD5_Base64(t *testing.T) {
	in := cryptoutil.HashInput{Input: "test", Algorithm: "md5", Encoding: "base64"}
	result := cryptoutil.Hash(context.Background(), in)

	var out cryptoutil.HashOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.Algorithm != "md5" {
		t.Errorf("wrong algorithm: %q", out.Algorithm)
	}
	if out.Hash == "" {
		t.Error("expected non-empty hash")
	}
}

func TestHash_SHA512_HappyPath(t *testing.T) {
	in := cryptoutil.HashInput{Input: "world", Algorithm: "sha512", Encoding: "hex"}
	result := cryptoutil.Hash(context.Background(), in)

	var out cryptoutil.HashOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(out.Hash) != 128 { // SHA-512 = 64 bytes = 128 hex chars
		t.Errorf("expected 128 hex chars, got %d", len(out.Hash))
	}
}

func TestHash_EmptyInput_ReturnsError(t *testing.T) {
	in := cryptoutil.HashInput{Input: "", Algorithm: "sha256"}
	result := cryptoutil.Hash(context.Background(), in)

	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := errOut["error"]; !ok {
		t.Errorf("expected error response, got: %s", result)
	}
}

func TestHash_UnknownAlgorithm_ReturnsError(t *testing.T) {
	in := cryptoutil.HashInput{Input: "hello", Algorithm: "blake2b"}
	result := cryptoutil.Hash(context.Background(), in)

	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := errOut["error"]; !ok {
		t.Errorf("expected error response, got: %s", result)
	}
}

// ── crypto_hmac ──────────────────────────────────────────────────────────────

func TestHMAC_SHA256_HappyPath(t *testing.T) {
	in := cryptoutil.HMACInput{
		Message:   "message",
		Key:       "secret",
		Algorithm: "sha256",
		Encoding:  "hex",
	}
	result := cryptoutil.HMAC(context.Background(), in)

	var out cryptoutil.HMACOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.HMAC == "" {
		t.Error("expected non-empty HMAC")
	}
	if out.Algorithm != "sha256" {
		t.Errorf("wrong algorithm: %q", out.Algorithm)
	}
	// SHA-256 HMAC = 32 bytes = 64 hex chars
	if len(out.HMAC) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(out.HMAC))
	}
}

func TestHMAC_SHA512_Base64(t *testing.T) {
	in := cryptoutil.HMACInput{
		Message:   "data",
		Key:       "key",
		Algorithm: "sha512",
		Encoding:  "base64",
	}
	result := cryptoutil.HMAC(context.Background(), in)

	var out cryptoutil.HMACOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out.HMAC == "" {
		t.Error("expected non-empty HMAC")
	}
}

func TestHMAC_MissingMessage_ReturnsError(t *testing.T) {
	in := cryptoutil.HMACInput{Message: "", Key: "secret"}
	result := cryptoutil.HMAC(context.Background(), in)

	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := errOut["error"]; !ok {
		t.Errorf("expected error response, got: %s", result)
	}
}

func TestHMAC_MissingKey_ReturnsError(t *testing.T) {
	in := cryptoutil.HMACInput{Message: "hello", Key: ""}
	result := cryptoutil.HMAC(context.Background(), in)

	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := errOut["error"]; !ok {
		t.Errorf("expected error response, got: %s", result)
	}
}

// ── jwt ─────────────────────────────────────────────────────────────────────

func TestJWT_Generate_HS256(t *testing.T) {
	in := cryptoutil.JWTInput{
		Operation:     "generate",
		Secret:        "mysecret",
		Payload:       `{"sub":"user123"}`,
		ExpirySeconds: 3600,
		Algorithm:     "HS256",
	}
	result := cryptoutil.JWT(context.Background(), in)

	var out map[string]string
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	token, ok := out["token"]
	if !ok || token == "" {
		t.Fatalf("expected token, got: %s", result)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("expected 3 JWT parts, got %d", len(parts))
	}
}

func TestJWT_Decode_ValidToken(t *testing.T) {
	// First generate a token
	genIn := cryptoutil.JWTInput{
		Operation:     "generate",
		Secret:        "secret",
		Payload:       `{"role":"admin"}`,
		ExpirySeconds: 3600,
		Algorithm:     "HS256",
	}
	genResult := cryptoutil.JWT(context.Background(), genIn)
	var genOut map[string]string
	json.Unmarshal([]byte(genResult), &genOut)
	token := genOut["token"]

	// Now decode it
	decIn := cryptoutil.JWTInput{Operation: "decode", Token: token}
	decResult := cryptoutil.JWT(context.Background(), decIn)

	var out map[string]any
	if err := json.Unmarshal([]byte(decResult), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, decResult)
	}
	if _, ok := out["header"]; !ok {
		t.Error("expected header in decode result")
	}
	if _, ok := out["payload"]; !ok {
		t.Error("expected payload in decode result")
	}
	expired, _ := out["expired"].(bool)
	if expired {
		t.Error("token should not be expired immediately after generation")
	}
}

func TestJWT_Verify_ValidToken(t *testing.T) {
	secret := "verify-secret"
	genIn := cryptoutil.JWTInput{
		Operation:     "generate",
		Secret:        secret,
		Payload:       `{"user":"alice"}`,
		ExpirySeconds: 3600,
		Algorithm:     "HS256",
	}
	genResult := cryptoutil.JWT(context.Background(), genIn)
	var genOut map[string]string
	json.Unmarshal([]byte(genResult), &genOut)
	token := genOut["token"]

	verifyIn := cryptoutil.JWTInput{Operation: "verify", Token: token, Secret: secret, Algorithm: "HS256"}
	verifyResult := cryptoutil.JWT(context.Background(), verifyIn)

	var out map[string]any
	if err := json.Unmarshal([]byte(verifyResult), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, verifyResult)
	}
	valid, _ := out["valid"].(bool)
	if !valid {
		t.Errorf("expected valid=true, got: %s", verifyResult)
	}
}

func TestJWT_Verify_WrongSecret_Invalid(t *testing.T) {
	genIn := cryptoutil.JWTInput{
		Operation: "generate",
		Secret:    "real-secret",
		Payload:   `{"x":1}`,
		Algorithm: "HS256",
	}
	genResult := cryptoutil.JWT(context.Background(), genIn)
	var genOut map[string]string
	json.Unmarshal([]byte(genResult), &genOut)

	verifyIn := cryptoutil.JWTInput{
		Operation: "verify",
		Token:     genOut["token"],
		Secret:    "wrong-secret",
		Algorithm: "HS256",
	}
	verifyResult := cryptoutil.JWT(context.Background(), verifyIn)

	var out map[string]any
	if err := json.Unmarshal([]byte(verifyResult), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	valid, _ := out["valid"].(bool)
	if valid {
		t.Error("expected valid=false with wrong secret")
	}
}

func TestJWT_MissingOperation_ReturnsError(t *testing.T) {
	in := cryptoutil.JWTInput{Token: "some.token.here"}
	result := cryptoutil.JWT(context.Background(), in)

	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := errOut["error"]; !ok {
		t.Errorf("expected error response, got: %s", result)
	}
}

func TestJWT_Decode_InvalidFormat_ReturnsError(t *testing.T) {
	in := cryptoutil.JWTInput{Operation: "decode", Token: "notavalidjwt"}
	result := cryptoutil.JWT(context.Background(), in)

	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := errOut["error"]; !ok {
		t.Errorf("expected error response, got: %s", result)
	}
}

// ── crypto_password ──────────────────────────────────────────────────────────

func TestPassword_BCrypt_HashAndVerify(t *testing.T) {
	// Hash
	hashIn := cryptoutil.PasswordInput{
		Password:  "supersecret",
		Operation: "hash",
		Algorithm: "bcrypt",
		Cost:      4, // use minimum cost for fast tests
	}
	hashResult := cryptoutil.Password(context.Background(), hashIn)

	var hashOut map[string]string
	if err := json.Unmarshal([]byte(hashResult), &hashOut); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, hashResult)
	}
	h, ok := hashOut["hash"]
	if !ok || h == "" {
		t.Fatalf("expected hash in response, got: %s", hashResult)
	}

	// Verify correct password
	verifyIn := cryptoutil.PasswordInput{
		Password:  "supersecret",
		Operation: "verify",
		Algorithm: "bcrypt",
		Hash:      h,
	}
	verifyResult := cryptoutil.Password(context.Background(), verifyIn)
	var verifyOut map[string]bool
	if err := json.Unmarshal([]byte(verifyResult), &verifyOut); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, verifyResult)
	}
	if !verifyOut["valid"] {
		t.Error("expected valid=true for correct password")
	}
}

func TestPassword_BCrypt_WrongPassword_Invalid(t *testing.T) {
	hashIn := cryptoutil.PasswordInput{
		Password:  "correct",
		Operation: "hash",
		Algorithm: "bcrypt",
		Cost:      4,
	}
	hashResult := cryptoutil.Password(context.Background(), hashIn)
	var hashOut map[string]string
	json.Unmarshal([]byte(hashResult), &hashOut)

	verifyIn := cryptoutil.PasswordInput{
		Password:  "wrong",
		Operation: "verify",
		Algorithm: "bcrypt",
		Hash:      hashOut["hash"],
	}
	verifyResult := cryptoutil.Password(context.Background(), verifyIn)
	var verifyOut map[string]bool
	json.Unmarshal([]byte(verifyResult), &verifyOut)
	if verifyOut["valid"] {
		t.Error("expected valid=false for wrong password")
	}
}

func TestPassword_Argon2id_HashAndVerify(t *testing.T) {
	hashIn := cryptoutil.PasswordInput{
		Password:  "my-password",
		Operation: "hash",
		Algorithm: "argon2id",
	}
	hashResult := cryptoutil.Password(context.Background(), hashIn)

	var hashOut map[string]string
	if err := json.Unmarshal([]byte(hashResult), &hashOut); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, hashResult)
	}
	h := hashOut["hash"]
	if !strings.HasPrefix(h, "$argon2id$") {
		t.Errorf("expected argon2id hash prefix, got: %s", h)
	}

	// Verify
	verifyIn := cryptoutil.PasswordInput{
		Password:  "my-password",
		Operation: "verify",
		Algorithm: "argon2id",
		Hash:      h,
	}
	verifyResult := cryptoutil.Password(context.Background(), verifyIn)
	var verifyOut map[string]bool
	if err := json.Unmarshal([]byte(verifyResult), &verifyOut); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, verifyResult)
	}
	if !verifyOut["valid"] {
		t.Error("expected valid=true for correct argon2id password")
	}
}

func TestPassword_MissingPassword_ReturnsError(t *testing.T) {
	in := cryptoutil.PasswordInput{Password: "", Operation: "hash"}
	result := cryptoutil.Password(context.Background(), in)

	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := errOut["error"]; !ok {
		t.Errorf("expected error response, got: %s", result)
	}
}

// ── crypto_keygen ─────────────────────────────────────────────────────────────

func TestKeygen_RSA_PEM(t *testing.T) {
	in := cryptoutil.KeygenInput{KeyType: "rsa", Bits: 2048, Format: "pem"}
	result := cryptoutil.Keygen(context.Background(), in)

	var out cryptoutil.KeygenOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if !strings.Contains(out.PrivateKey, "PRIVATE KEY") {
		t.Errorf("expected PEM private key, got: %s", out.PrivateKey[:min(50, len(out.PrivateKey))])
	}
	if !strings.Contains(out.PublicKey, "PUBLIC KEY") {
		t.Errorf("expected PEM public key, got: %s", out.PublicKey[:min(50, len(out.PublicKey))])
	}
	if out.KeyType != "rsa" {
		t.Errorf("wrong key_type: %q", out.KeyType)
	}
}

func TestKeygen_EC_P256_PEM(t *testing.T) {
	in := cryptoutil.KeygenInput{KeyType: "ec", Curve: "P-256", Format: "pem"}
	result := cryptoutil.Keygen(context.Background(), in)

	var out cryptoutil.KeygenOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if !strings.Contains(out.PrivateKey, "EC PRIVATE KEY") {
		t.Errorf("expected EC PEM private key, got: %s", out.PrivateKey[:min(50, len(out.PrivateKey))])
	}
	if out.KeyType != "ec" {
		t.Errorf("wrong key_type: %q", out.KeyType)
	}
}

func TestKeygen_ED25519_PEM(t *testing.T) {
	in := cryptoutil.KeygenInput{KeyType: "ed25519", Format: "pem"}
	result := cryptoutil.Keygen(context.Background(), in)

	var out cryptoutil.KeygenOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if !strings.Contains(out.PrivateKey, "PRIVATE KEY") {
		t.Errorf("expected PEM private key, got: %s", out.PrivateKey[:min(50, len(out.PrivateKey))])
	}
	if out.KeyType != "ed25519" {
		t.Errorf("wrong key_type: %q", out.KeyType)
	}
}

func TestKeygen_RSA_JWK(t *testing.T) {
	in := cryptoutil.KeygenInput{KeyType: "rsa", Bits: 2048, Format: "jwk"}
	result := cryptoutil.Keygen(context.Background(), in)

	var out cryptoutil.KeygenOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	// Private JWK must have "kty":"RSA"
	var privJWK map[string]any
	if err := json.Unmarshal([]byte(out.PrivateKey), &privJWK); err != nil {
		t.Fatalf("private key is not valid JSON: %v", err)
	}
	if privJWK["kty"] != "RSA" {
		t.Errorf("expected kty=RSA in JWK, got: %v", privJWK["kty"])
	}
}

func TestKeygen_EC_JWK(t *testing.T) {
	in := cryptoutil.KeygenInput{KeyType: "ec", Curve: "P-384", Format: "jwk"}
	result := cryptoutil.Keygen(context.Background(), in)

	var out cryptoutil.KeygenOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	var pubJWK map[string]any
	if err := json.Unmarshal([]byte(out.PublicKey), &pubJWK); err != nil {
		t.Fatalf("public key is not valid JSON: %v", err)
	}
	if pubJWK["kty"] != "EC" {
		t.Errorf("expected kty=EC in JWK, got: %v", pubJWK["kty"])
	}
	if pubJWK["crv"] != "P-384" {
		t.Errorf("expected crv=P-384, got: %v", pubJWK["crv"])
	}
}

func TestKeygen_MissingKeyType_ReturnsError(t *testing.T) {
	in := cryptoutil.KeygenInput{}
	result := cryptoutil.Keygen(context.Background(), in)

	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := errOut["error"]; !ok {
		t.Errorf("expected error response, got: %s", result)
	}
}

func TestKeygen_InvalidBits_ReturnsError(t *testing.T) {
	in := cryptoutil.KeygenInput{KeyType: "rsa", Bits: 512}
	result := cryptoutil.Keygen(context.Background(), in)

	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := errOut["error"]; !ok {
		t.Errorf("expected error response, got: %s", result)
	}
}

// ── crypto_random ─────────────────────────────────────────────────────────────

func TestRandom_Token_Hex(t *testing.T) {
	in := cryptoutil.RandomInput{Kind: "token", Length: 32, Encoding: "hex"}
	result := cryptoutil.Random(context.Background(), in)

	var out map[string]string
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	val := out["value"]
	if len(val) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("expected 64 hex chars, got %d", len(val))
	}
}

func TestRandom_Token_Base64(t *testing.T) {
	in := cryptoutil.RandomInput{Kind: "token", Length: 16, Encoding: "base64"}
	result := cryptoutil.Random(context.Background(), in)

	var out map[string]string
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out["value"] == "" {
		t.Error("expected non-empty value")
	}
}

func TestRandom_OTP_Length6(t *testing.T) {
	in := cryptoutil.RandomInput{Kind: "otp", Length: 6}
	result := cryptoutil.Random(context.Background(), in)

	var out map[string]string
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	val := out["value"]
	if len(val) != 6 {
		t.Errorf("expected 6-digit OTP, got %d chars: %s", len(val), val)
	}
	for _, c := range val {
		if c < '0' || c > '9' {
			t.Errorf("OTP contains non-digit character: %c", c)
		}
	}
}

func TestRandom_Bytes_Base64url(t *testing.T) {
	in := cryptoutil.RandomInput{Kind: "bytes", Length: 24, Encoding: "base64url"}
	result := cryptoutil.Random(context.Background(), in)

	var out map[string]string
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out["value"] == "" {
		t.Error("expected non-empty value")
	}
	// base64url must not contain + or /
	if strings.ContainsAny(out["value"], "+/") {
		t.Errorf("base64url should not contain + or /, got: %s", out["value"])
	}
}

func TestRandom_MissingKind_ReturnsError(t *testing.T) {
	in := cryptoutil.RandomInput{}
	result := cryptoutil.Random(context.Background(), in)

	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := errOut["error"]; !ok {
		t.Errorf("expected error response, got: %s", result)
	}
}

// ── crypto_mask ───────────────────────────────────────────────────────────────

func TestMask_APIKey_Redacted(t *testing.T) {
	text := `Use token sk-abcdefghijklmnopqrstuvwxyz1234 to authenticate`
	in := cryptoutil.MaskInput{
		Text:        text,
		Patterns:    []string{"api_key"},
		Replacement: "[REDACTED]",
	}
	result := cryptoutil.Mask(context.Background(), in)

	var out cryptoutil.MaskOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if strings.Contains(out.Result, "sk-abcdefghijklmnopqrstuvwxyz1234") {
		t.Error("API key should be redacted")
	}
	if out.RedactedCount < 1 {
		t.Errorf("expected at least 1 redaction, got %d", out.RedactedCount)
	}
}

func TestMask_Email_Redacted(t *testing.T) {
	text := `Contact user@example.com for details`
	in := cryptoutil.MaskInput{
		Text:     text,
		Patterns: []string{"email"},
	}
	result := cryptoutil.Mask(context.Background(), in)

	var out cryptoutil.MaskOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if strings.Contains(out.Result, "user@example.com") {
		t.Error("email should be redacted")
	}
}

func TestMask_JWT_Redacted(t *testing.T) {
	// A real-looking JWT
	jwtToken := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c2VyMTIzIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	text := "Authorization: Bearer " + jwtToken
	in := cryptoutil.MaskInput{
		Text:     text,
		Patterns: []string{"jwt"},
	}
	result := cryptoutil.Mask(context.Background(), in)

	var out cryptoutil.MaskOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if strings.Contains(out.Result, jwtToken) {
		t.Error("JWT should be redacted")
	}
}

func TestMask_All_MultiplePatterns(t *testing.T) {
	text := `email: admin@test.com, key: sk-secretkeyhere12345678901234, pass: password=hunter2`
	in := cryptoutil.MaskInput{
		Text:     text,
		Patterns: []string{"all"},
	}
	result := cryptoutil.Mask(context.Background(), in)

	var out cryptoutil.MaskOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.RedactedCount < 2 {
		t.Errorf("expected at least 2 redactions with 'all' pattern, got %d — result: %s", out.RedactedCount, out.Result)
	}
}

func TestMask_EmptyText_ReturnsError(t *testing.T) {
	in := cryptoutil.MaskInput{Text: ""}
	result := cryptoutil.Mask(context.Background(), in)

	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := errOut["error"]; !ok {
		t.Errorf("expected error response, got: %s", result)
	}
}

func TestMask_NoSensitiveData_ZeroRedactions(t *testing.T) {
	in := cryptoutil.MaskInput{
		Text:     "Hello, this is a plain text with no secrets.",
		Patterns: []string{"api_key"},
	}
	result := cryptoutil.Mask(context.Background(), in)

	var out cryptoutil.MaskOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out.RedactedCount != 0 {
		t.Errorf("expected 0 redactions, got %d", out.RedactedCount)
	}
}

// min is a helper for slicing in tests (Go 1.21 has a built-in, but this is safe).
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── password_generate tests ──────────────────────────────────────────────────

func TestPasswordGenerate_DefaultParams(t *testing.T) {
	in := cryptoutil.PasswordGenerateInput{
		Length:           16,
		IncludeUppercase: true,
		IncludeLowercase: true,
		IncludeNumbers:   true,
		IncludeSymbols:   true,
	}
	result := cryptoutil.PasswordGenerate(context.Background(), in)
	var out cryptoutil.PasswordGenerateOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if len(out.Password) != 16 {
		t.Errorf("expected password length 16, got %d", len(out.Password))
	}
	if out.Length != 16 {
		t.Errorf("expected length field 16, got %d", out.Length)
	}
	if out.Entropy <= 0 {
		t.Errorf("expected positive entropy, got %f", out.Entropy)
	}
}

func TestPasswordGenerate_CustomLength(t *testing.T) {
	in := cryptoutil.PasswordGenerateInput{
		Length:           32,
		IncludeUppercase: true,
		IncludeLowercase: true,
		IncludeNumbers:   true,
		IncludeSymbols:   false,
	}
	result := cryptoutil.PasswordGenerate(context.Background(), in)
	var out cryptoutil.PasswordGenerateOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if len(out.Password) != 32 {
		t.Errorf("expected password length 32, got %d", len(out.Password))
	}
}

func TestPasswordGenerate_OnlyLowercase(t *testing.T) {
	in := cryptoutil.PasswordGenerateInput{
		Length:           20,
		IncludeUppercase: false,
		IncludeLowercase: true,
		IncludeNumbers:   false,
		IncludeSymbols:   false,
	}
	result := cryptoutil.PasswordGenerate(context.Background(), in)
	var out cryptoutil.PasswordGenerateOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	// Should only contain lowercase letters
	for _, c := range out.Password {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyz", c) {
			t.Errorf("password contains non-lowercase char: %c", c)
		}
	}
}

func TestPasswordGenerate_OnlyNumbers(t *testing.T) {
	in := cryptoutil.PasswordGenerateInput{
		Length:           10,
		IncludeUppercase: false,
		IncludeLowercase: false,
		IncludeNumbers:   true,
		IncludeSymbols:   false,
	}
	result := cryptoutil.PasswordGenerate(context.Background(), in)
	var out cryptoutil.PasswordGenerateOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	// Should only contain digits
	for _, c := range out.Password {
		if !strings.ContainsRune("0123456789", c) {
			t.Errorf("password contains non-digit char: %c", c)
		}
	}
}

func TestPasswordGenerate_InvalidLength_ReturnsError(t *testing.T) {
	in := cryptoutil.PasswordGenerateInput{
		Length:           0,
		IncludeUppercase: true,
		IncludeLowercase: true,
		IncludeNumbers:   true,
		IncludeSymbols:   true,
	}
	result := cryptoutil.PasswordGenerate(context.Background(), in)
	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("expected error JSON, got: %s", result)
	}
	if _, ok := errOut["error"]; !ok {
		t.Error("expected error for invalid length")
	}
}

func TestPasswordGenerate_LengthTooLong_ReturnsError(t *testing.T) {
	in := cryptoutil.PasswordGenerateInput{
		Length:           51,
		IncludeUppercase: true,
		IncludeLowercase: true,
		IncludeNumbers:   true,
		IncludeSymbols:   true,
	}
	result := cryptoutil.PasswordGenerate(context.Background(), in)
	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("expected error JSON, got: %s", result)
	}
	if _, ok := errOut["error"]; !ok {
		t.Error("expected error for length > 50")
	}
}

func TestPasswordGenerate_NoCharacterSets_ReturnsError(t *testing.T) {
	in := cryptoutil.PasswordGenerateInput{
		Length:           16,
		IncludeUppercase: false,
		IncludeLowercase: false,
		IncludeNumbers:   false,
		IncludeSymbols:   false,
	}
	result := cryptoutil.PasswordGenerate(context.Background(), in)
	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("expected error JSON, got: %s", result)
	}
	if _, ok := errOut["error"]; !ok {
		t.Error("expected error when no character sets selected")
	}
}

func TestPasswordGenerate_Uniqueness(t *testing.T) {
	in := cryptoutil.PasswordGenerateInput{
		Length:           32,
		IncludeUppercase: true,
		IncludeLowercase: true,
		IncludeNumbers:   true,
		IncludeSymbols:   true,
	}
	passwords := make(map[string]bool)
	for i := 0; i < 100; i++ {
		result := cryptoutil.PasswordGenerate(context.Background(), in)
		var out cryptoutil.PasswordGenerateOutput
		json.Unmarshal([]byte(result), &out)
		if passwords[out.Password] {
			t.Errorf("duplicate password generated: %s", out.Password)
		}
		passwords[out.Password] = true
	}
}
