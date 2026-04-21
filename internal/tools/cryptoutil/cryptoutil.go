// Package cryptoutil implements MCP tools for cryptographic operations:
// hashing, HMAC, JWT (HS256/HS512), password hashing, key generation,
// random value generation, secret masking, and password generation.
package cryptoutil

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
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

// ── crypto_hash ──────────────────────────────────────────────────────────────

// HashInput is the input schema for crypto_hash.
type HashInput struct {
	Input     string `json:"input"`
	Algorithm string `json:"algorithm"` // sha256 | sha512 | md5 | sha1
	Encoding  string `json:"encoding"`  // hex | base64
}

// HashOutput is the output schema for crypto_hash.
type HashOutput struct {
	Hash      string `json:"hash"`
	Algorithm string `json:"algorithm"`
	Encoding  string `json:"encoding"`
}

// Hash computes a hash of the input string using the specified algorithm and encoding.
func Hash(_ context.Context, in HashInput) string {
	if in.Input == "" {
		return errResult("input is required")
	}
	algo := strings.ToLower(in.Algorithm)
	if algo == "" {
		algo = "sha256"
	}
	enc := strings.ToLower(in.Encoding)
	if enc == "" {
		enc = "hex"
	}

	var raw []byte
	switch algo {
	case "sha256":
		h := sha256.Sum256([]byte(in.Input))
		raw = h[:]
	case "sha512":
		h := sha512.Sum512([]byte(in.Input))
		raw = h[:]
	case "md5":
		h := md5.Sum([]byte(in.Input)) // MD5 exposed by explicit user choice
		raw = h[:]
	case "sha1":
		h := sha1.Sum([]byte(in.Input)) // SHA1 exposed by explicit user choice
		raw = h[:]
	default:
		return errResult("unsupported algorithm: " + algo + " (sha256|sha512|md5|sha1)")
	}

	encoded, err := encodeBytes(raw, enc)
	if err != nil {
		return errResult(err.Error())
	}

	return resultJSON(HashOutput{Hash: encoded, Algorithm: algo, Encoding: enc})
}

// ── crypto_hmac ──────────────────────────────────────────────────────────────

// HMACInput is the input schema for crypto_hmac.
type HMACInput struct {
	Message   string `json:"message"`
	Key       string `json:"key"`
	Algorithm string `json:"algorithm"` // sha256 | sha512
	Encoding  string `json:"encoding"`  // hex | base64
}

// HMACOutput is the output schema for crypto_hmac.
type HMACOutput struct {
	HMAC      string `json:"hmac"`
	Algorithm string `json:"algorithm"`
}

// HMAC computes an HMAC over the message using the given key.
func HMAC(_ context.Context, in HMACInput) string {
	if in.Message == "" {
		return errResult("message is required")
	}
	if in.Key == "" {
		return errResult("key is required")
	}

	algo := strings.ToLower(in.Algorithm)
	if algo == "" {
		algo = "sha256"
	}
	enc := strings.ToLower(in.Encoding)
	if enc == "" {
		enc = "hex"
	}

	var mac []byte
	switch algo {
	case "sha256":
		h := hmac.New(sha256.New, []byte(in.Key))
		h.Write([]byte(in.Message))
		mac = h.Sum(nil)
	case "sha512":
		h := hmac.New(sha512.New, []byte(in.Key))
		h.Write([]byte(in.Message))
		mac = h.Sum(nil)
	default:
		return errResult("unsupported algorithm: " + algo + " (sha256|sha512)")
	}

	encoded, err := encodeBytes(mac, enc)
	if err != nil {
		return errResult(err.Error())
	}

	return resultJSON(HMACOutput{HMAC: encoded, Algorithm: algo})
}

// ── jwt ─────────────────────────────────────────────────────────────────────

// JWTInput is the input schema for jwt.
type JWTInput struct {
	Token         string `json:"token"`
	Operation     string `json:"operation"`      // decode | verify | generate
	Secret        string `json:"secret"`         // for verify/generate
	Payload       string `json:"payload"`        // JSON object string, for generate
	ExpirySeconds int    `json:"expiry_seconds"` // default 3600
	Algorithm     string `json:"algorithm"`      // HS256 | HS512
}

// JWT processes a JWT token: decode, verify, or generate.
func JWT(_ context.Context, in JWTInput) string {
	op := strings.ToLower(in.Operation)
	if op == "" {
		return errResult("operation is required (decode|verify|generate)")
	}

	algo := strings.ToUpper(in.Algorithm)
	if algo == "" {
		algo = "HS256"
	}
	if algo != "HS256" && algo != "HS512" {
		return errResult("unsupported algorithm: " + algo + " (HS256|HS512)")
	}

	switch op {
	case "generate":
		return jwtGenerate(in, algo)
	case "decode":
		return jwtDecode(in.Token)
	case "verify":
		return jwtVerify(in.Token, in.Secret, algo)
	default:
		return errResult("unknown operation: " + op + " (decode|verify|generate)")
	}
}

func jwtGenerate(in JWTInput, algo string) string {
	if in.Secret == "" {
		return errResult("secret is required for generate")
	}

	// Build header
	header := map[string]string{"alg": algo, "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return errResult("failed to marshal header: " + err.Error())
	}

	// Build payload
	var claims map[string]any
	if in.Payload != "" {
		if err := json.Unmarshal([]byte(in.Payload), &claims); err != nil {
			return errResult("invalid payload JSON: " + err.Error())
		}
	} else {
		claims = make(map[string]any)
	}

	expiry := in.ExpirySeconds
	if expiry <= 0 {
		expiry = 3600
	}
	now := time.Now().Unix()
	claims["iat"] = now
	claims["exp"] = now + int64(expiry)

	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return errResult("failed to marshal payload: " + err.Error())
	}

	headerEnc := base64url(headerJSON)
	payloadEnc := base64url(payloadJSON)
	sigInput := headerEnc + "." + payloadEnc

	sig := jwtSign(sigInput, in.Secret, algo)
	token := sigInput + "." + sig

	return resultJSON(map[string]string{"token": token})
}

func jwtDecode(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return errResult("invalid JWT format: expected 3 parts")
	}

	headerJSON, err := base64urlDecode(parts[0])
	if err != nil {
		return errResult("invalid header encoding: " + err.Error())
	}
	payloadJSON, err := base64urlDecode(parts[1])
	if err != nil {
		return errResult("invalid payload encoding: " + err.Error())
	}

	var headerMap, payloadMap map[string]any
	if err := json.Unmarshal(headerJSON, &headerMap); err != nil {
		return errResult("invalid header JSON: " + err.Error())
	}
	if err := json.Unmarshal(payloadJSON, &payloadMap); err != nil {
		return errResult("invalid payload JSON: " + err.Error())
	}

	expired := false
	if exp, ok := payloadMap["exp"]; ok {
		var expUnix float64
		switch v := exp.(type) {
		case float64:
			expUnix = v
		case int64:
			expUnix = float64(v)
		}
		if expUnix > 0 && time.Now().Unix() > int64(expUnix) {
			expired = true
		}
	}

	return resultJSON(map[string]any{
		"header":    headerMap,
		"payload":   payloadMap,
		"signature": parts[2],
		"expired":   expired,
	})
}

func jwtVerify(token, secret, algo string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return errResult("invalid JWT format: expected 3 parts")
	}

	sigInput := parts[0] + "." + parts[1]
	expectedSig := jwtSign(sigInput, secret, algo)

	payloadJSON, err := base64urlDecode(parts[1])
	if err != nil {
		return errResult("invalid payload encoding: " + err.Error())
	}
	var claims map[string]any
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return errResult("invalid payload JSON: " + err.Error())
	}

	expired := false
	if exp, ok := claims["exp"]; ok {
		var expUnix float64
		switch v := exp.(type) {
		case float64:
			expUnix = v
		case int64:
			expUnix = float64(v)
		}
		if expUnix > 0 && time.Now().Unix() > int64(expUnix) {
			expired = true
		}
	}

	valid := hmac.Equal([]byte(parts[2]), []byte(expectedSig)) && !expired

	return resultJSON(map[string]any{
		"valid":   valid,
		"expired": expired,
		"claims":  claims,
	})
}

// jwtSign computes the HMAC signature for a JWT (header.payload) string.
func jwtSign(input, secret, algo string) string {
	var mac []byte
	switch algo {
	case "HS512":
		h := hmac.New(sha512.New, []byte(secret))
		h.Write([]byte(input))
		mac = h.Sum(nil)
	default: // HS256
		h := hmac.New(sha256.New, []byte(secret))
		h.Write([]byte(input))
		mac = h.Sum(nil)
	}
	return base64url(mac)
}

// base64url encodes bytes using base64 URL encoding without padding.
func base64url(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// base64urlDecode decodes a base64url string (with or without padding).
func base64urlDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// ── crypto_password ──────────────────────────────────────────────────────────

// PasswordInput is the input schema for crypto_password.
type PasswordInput struct {
	Password  string `json:"password"`
	Operation string `json:"operation"` // hash | verify
	Algorithm string `json:"algorithm"` // bcrypt | argon2id
	Hash      string `json:"hash"`      // required for verify
	Cost      int    `json:"cost"`      // bcrypt cost, default 12
}

// Password hashes or verifies a password using bcrypt or argon2id.
func Password(_ context.Context, in PasswordInput) string {
	if in.Password == "" {
		return errResult("password is required")
	}
	op := strings.ToLower(in.Operation)
	if op == "" {
		return errResult("operation is required (hash|verify)")
	}
	algo := strings.ToLower(in.Algorithm)
	if algo == "" {
		algo = "bcrypt"
	}

	switch op {
	case "hash":
		return passwordHash(in.Password, algo, in.Cost)
	case "verify":
		if in.Hash == "" {
			return errResult("hash is required for verify")
		}
		return passwordVerify(in.Password, in.Hash, algo)
	default:
		return errResult("unknown operation: " + op + " (hash|verify)")
	}
}

func passwordHash(password, algo string, cost int) string {
	switch algo {
	case "bcrypt":
		if cost <= 0 {
			cost = 12
		}
		if cost < bcrypt.MinCost {
			cost = bcrypt.MinCost
		}
		if cost > bcrypt.MaxCost {
			cost = bcrypt.MaxCost
		}
		h, err := bcrypt.GenerateFromPassword([]byte(password), cost)
		if err != nil {
			return errResult("bcrypt hash failed: " + err.Error())
		}
		return resultJSON(map[string]string{"hash": string(h)})

	case "argon2id":
		salt := make([]byte, 16)
		if _, err := rand.Read(salt); err != nil {
			return errResult("failed to generate salt: " + err.Error())
		}
		// Argon2id recommended parameters: time=1, memory=64MB, threads=4, keyLen=32
		key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
		encoded := fmt.Sprintf(
			"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
			argon2.Version,
			64*1024,
			1,
			4,
			base64.RawStdEncoding.EncodeToString(salt),
			base64.RawStdEncoding.EncodeToString(key),
		)
		return resultJSON(map[string]string{"hash": encoded})

	default:
		return errResult("unsupported algorithm: " + algo + " (bcrypt|argon2id)")
	}
}

func passwordVerify(password, hash, algo string) string {
	switch algo {
	case "bcrypt":
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		return resultJSON(map[string]bool{"valid": err == nil})

	case "argon2id":
		valid, err := verifyArgon2id(password, hash)
		if err != nil {
			return errResult("argon2id verify failed: " + err.Error())
		}
		return resultJSON(map[string]bool{"valid": valid})

	default:
		return errResult("unsupported algorithm: " + algo + " (bcrypt|argon2id)")
	}
}

// verifyArgon2id parses a PHC-format argon2id hash and verifies the password.
func verifyArgon2id(password, encoded string) (bool, error) {
	// Format: $argon2id$v=N$m=M,t=T,p=P$salt$hash
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, fmt.Errorf("invalid argon2id hash format")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("invalid version: %w", err)
	}

	var memory, timeCost, parallelism uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeCost, &parallelism); err != nil {
		return false, fmt.Errorf("invalid params: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("invalid salt: %w", err)
	}
	hashBytes, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("invalid hash: %w", err)
	}

	computed := argon2.IDKey([]byte(password), salt, timeCost, memory, uint8(parallelism), uint32(len(hashBytes)))
	return hmac.Equal(computed, hashBytes), nil
}

// ── crypto_keygen ─────────────────────────────────────────────────────────────

// KeygenInput is the input schema for crypto_keygen.
type KeygenInput struct {
	KeyType string `json:"key_type"` // rsa | ec | ed25519
	Bits    int    `json:"bits"`     // 2048 | 4096 for RSA
	Curve   string `json:"curve"`    // P-256 | P-384 for EC
	Format  string `json:"format"`   // pem | jwk
}

// KeygenOutput is the output schema for crypto_keygen.
type KeygenOutput struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	KeyType    string `json:"key_type"`
}

// Keygen generates a cryptographic key pair.
func Keygen(_ context.Context, in KeygenInput) string {
	kt := strings.ToLower(in.KeyType)
	if kt == "" {
		return errResult("key_type is required (rsa|ec|ed25519)")
	}

	format := strings.ToLower(in.Format)
	if format == "" {
		format = "pem"
	}
	if format != "pem" && format != "jwk" {
		return errResult("unsupported format: " + format + " (pem|jwk)")
	}

	switch kt {
	case "rsa":
		return keygenRSA(in.Bits, format)
	case "ec":
		return keygenEC(in.Curve, format)
	case "ed25519":
		return keygenED25519(format)
	default:
		return errResult("unsupported key_type: " + kt + " (rsa|ec|ed25519)")
	}
}

func keygenRSA(bits int, format string) string {
	if bits == 0 {
		bits = 2048
	}
	if bits != 2048 && bits != 4096 {
		return errResult("bits must be 2048 or 4096 for RSA")
	}

	priv, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return errResult("RSA key generation failed: " + err.Error())
	}

	if format == "jwk" {
		return keygenRSAJWK(priv)
	}

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return errResult("failed to marshal RSA private key: " + err.Error())
	}
	privPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}))

	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return errResult("failed to marshal RSA public key: " + err.Error())
	}
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))

	return resultJSON(KeygenOutput{PrivateKey: privPEM, PublicKey: pubPEM, KeyType: "rsa"})
}

func keygenRSAJWK(priv *rsa.PrivateKey) string {
	pub := &priv.PublicKey
	// Build JWK manually (public + private)
	privJWK := map[string]any{
		"kty": "RSA",
		"n":   base64url(pub.N.Bytes()),
		"e":   base64url(big.NewInt(int64(pub.E)).Bytes()),
		"d":   base64url(priv.D.Bytes()),
		"p":   base64url(priv.Primes[0].Bytes()),
		"q":   base64url(priv.Primes[1].Bytes()),
		"dp":  base64url(priv.Precomputed.Dp.Bytes()),
		"dq":  base64url(priv.Precomputed.Dq.Bytes()),
		"qi":  base64url(priv.Precomputed.Qinv.Bytes()),
	}
	pubJWK := map[string]any{
		"kty": "RSA",
		"n":   base64url(pub.N.Bytes()),
		"e":   base64url(big.NewInt(int64(pub.E)).Bytes()),
	}
	privJSON, _ := json.Marshal(privJWK)
	pubJSON, _ := json.Marshal(pubJWK)
	return resultJSON(KeygenOutput{
		PrivateKey: string(privJSON),
		PublicKey:  string(pubJSON),
		KeyType:    "rsa",
	})
}

func keygenEC(curve, format string) string {
	if curve == "" {
		curve = "P-256"
	}
	var c elliptic.Curve
	switch curve {
	case "P-256":
		c = elliptic.P256()
	case "P-384":
		c = elliptic.P384()
	default:
		return errResult("unsupported curve: " + curve + " (P-256|P-384)")
	}

	priv, err := ecdsa.GenerateKey(c, rand.Reader)
	if err != nil {
		return errResult("EC key generation failed: " + err.Error())
	}

	if format == "jwk" {
		return keygenECJWK(priv, curve)
	}

	privDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return errResult("failed to marshal EC private key: " + err.Error())
	}
	privPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER}))

	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return errResult("failed to marshal EC public key: " + err.Error())
	}
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))

	return resultJSON(KeygenOutput{PrivateKey: privPEM, PublicKey: pubPEM, KeyType: "ec"})
}

func keygenECJWK(priv *ecdsa.PrivateKey, curve string) string {
	privJWK := map[string]any{
		"kty": "EC",
		"crv": curve,
		"x":   base64url(priv.PublicKey.X.Bytes()),
		"y":   base64url(priv.PublicKey.Y.Bytes()),
		"d":   base64url(priv.D.Bytes()),
	}
	pubJWK := map[string]any{
		"kty": "EC",
		"crv": curve,
		"x":   base64url(priv.PublicKey.X.Bytes()),
		"y":   base64url(priv.PublicKey.Y.Bytes()),
	}
	privJSON, _ := json.Marshal(privJWK)
	pubJSON, _ := json.Marshal(pubJWK)
	return resultJSON(KeygenOutput{
		PrivateKey: string(privJSON),
		PublicKey:  string(pubJSON),
		KeyType:    "ec",
	})
}

func keygenED25519(format string) string {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return errResult("Ed25519 key generation failed: " + err.Error())
	}

	if format == "jwk" {
		privJWK := map[string]any{
			"kty": "OKP",
			"crv": "Ed25519",
			"x":   base64url(pub),
			"d":   base64url(priv.Seed()),
		}
		pubJWK := map[string]any{
			"kty": "OKP",
			"crv": "Ed25519",
			"x":   base64url(pub),
		}
		privJSON, _ := json.Marshal(privJWK)
		pubJSON, _ := json.Marshal(pubJWK)
		return resultJSON(KeygenOutput{
			PrivateKey: string(privJSON),
			PublicKey:  string(pubJSON),
			KeyType:    "ed25519",
		})
	}

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return errResult("failed to marshal Ed25519 private key: " + err.Error())
	}
	privPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}))

	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return errResult("failed to marshal Ed25519 public key: " + err.Error())
	}
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))

	return resultJSON(KeygenOutput{PrivateKey: privPEM, PublicKey: pubPEM, KeyType: "ed25519"})
}

// ── crypto_random ─────────────────────────────────────────────────────────────

// RandomInput is the input schema for crypto_random.
type RandomInput struct {
	Kind     string `json:"kind"`     // token | bytes | otp
	Length   int    `json:"length"`   // default 32 (token/bytes), 6 (otp)
	Encoding string `json:"encoding"` // hex | base64 | base64url
}

// Random generates cryptographically secure random values.
func Random(_ context.Context, in RandomInput) string {
	kind := strings.ToLower(in.Kind)
	if kind == "" {
		return errResult("kind is required (token|bytes|otp)")
	}

	enc := strings.ToLower(in.Encoding)
	if enc == "" {
		enc = "hex"
	}

	switch kind {
	case "token", "bytes":
		length := in.Length
		if length <= 0 {
			length = 32
		}
		buf := make([]byte, length)
		if _, err := rand.Read(buf); err != nil {
			return errResult("random generation failed: " + err.Error())
		}
		encoded, err := encodeBytes(buf, enc)
		if err != nil {
			return errResult(err.Error())
		}
		return resultJSON(map[string]string{"value": encoded})

	case "otp":
		length := in.Length
		if length <= 0 {
			length = 6
		}
		digits, err := randomDigits(length)
		if err != nil {
			return errResult("OTP generation failed: " + err.Error())
		}
		return resultJSON(map[string]string{"value": digits})

	default:
		return errResult("unsupported kind: " + kind + " (token|bytes|otp)")
	}
}

// randomDigits generates a string of n random decimal digits using crypto/rand.
func randomDigits(n int) (string, error) {
	const digits = "0123456789"
	buf := make([]byte, n)
	for i := range buf {
		idx, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		buf[i] = digits[idx.Int64()]
	}
	return string(buf), nil
}

// ── crypto_mask ───────────────────────────────────────────────────────────────

// MaskInput is the input schema for crypto_mask.
type MaskInput struct {
	Text        string   `json:"text"`
	Patterns    []string `json:"patterns"`    // api_key | password | email | credit_card | jwt | all
	Replacement string   `json:"replacement"` // default [REDACTED]
}

// MaskOutput is the output schema for crypto_mask.
type MaskOutput struct {
	Result        string `json:"result"`
	RedactedCount int    `json:"redacted_count"`
}

// maskPattern pairs a human-readable name with its compiled regex.
type maskPattern struct {
	name string
	re   *regexp.Regexp
}

// maskPatterns holds all supported regex patterns for secret detection.
var maskPatterns = []maskPattern{
	// API keys — common prefixes (sk-, pk-, api_, key_) followed by 20+ word chars
	{name: "api_key", re: regexp.MustCompile(`(?i)(sk|pk|api|key)[_-][a-zA-Z0-9_\-]{20,}`)},
	// Password-looking key=value pairs
	{name: "password", re: regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*["']?[^\s"']{4,}["']?`)},
	// Email addresses
	{name: "email", re: regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)},
	// Credit card numbers (Luhn-compatible pattern: 13–19 digits, with optional separators)
	{name: "credit_card", re: regexp.MustCompile(`\b(?:\d[ -]?){13,19}\b`)},
	// JWT tokens (three base64url segments separated by dots)
	{name: "jwt", re: regexp.MustCompile(`eyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`)},
}

// Mask redacts sensitive patterns from text.
func Mask(_ context.Context, in MaskInput) string {
	if in.Text == "" {
		return errResult("text is required")
	}

	replacement := in.Replacement
	if replacement == "" {
		replacement = "[REDACTED]"
	}

	patterns := in.Patterns
	if len(patterns) == 0 {
		patterns = []string{"all"}
	}

	// Determine which patterns to activate
	activate := make(map[string]bool)
	for _, p := range patterns {
		if strings.ToLower(p) == "all" {
			for _, mp := range maskPatterns {
				activate[mp.name] = true
			}
		} else {
			activate[strings.ToLower(p)] = true
		}
	}

	result := in.Text
	redactedCount := 0

	for _, mp := range maskPatterns {
		if !activate[mp.name] {
			continue
		}
		matches := mp.re.FindAllString(result, -1)
		if len(matches) > 0 {
			redactedCount += len(matches)
			result = mp.re.ReplaceAllString(result, replacement)
		}
	}

	return resultJSON(MaskOutput{Result: result, RedactedCount: redactedCount})
}

// ── helpers ───────────────────────────────────────────────────────────────────

// encodeBytes encodes raw bytes as hex or base64.
func encodeBytes(raw []byte, encoding string) (string, error) {
	switch encoding {
	case "hex":
		return fmt.Sprintf("%x", raw), nil
	case "base64":
		return base64.StdEncoding.EncodeToString(raw), nil
	case "base64url":
		return base64.RawURLEncoding.EncodeToString(raw), nil
	default:
		return "", fmt.Errorf("unsupported encoding: %s (hex|base64|base64url)", encoding)
	}
}

// ─── password_generate ───────────────────────────────────────────────────────

const (
	uppercaseChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercaseChars = "abcdefghijklmnopqrstuvwxyz"
	numberChars    = "0123456789"
	symbolChars    = "!@#$%^&*()_+-=[]{}|;:,.<>?"
)

// PasswordGenerateInput is the input schema for the password_generate tool.
type PasswordGenerateInput struct {
	Length           int  `json:"length"`
	IncludeUppercase bool `json:"include_uppercase"`
	IncludeLowercase bool `json:"include_lowercase"`
	IncludeNumbers   bool `json:"include_numbers"`
	IncludeSymbols   bool `json:"include_symbols"`
}

// PasswordGenerateOutput is the output schema for the password_generate tool.
type PasswordGenerateOutput struct {
	Password string `json:"password"`
	Length   int    `json:"length"`
	Entropy  float64 `json:"entropy_bits"`
}

// PasswordGenerate creates a secure random password with configurable character sets.
func PasswordGenerate(_ context.Context, input PasswordGenerateInput) string {
	length := input.Length
	if length < 1 || length > 50 {
		return errResult("length must be between 1 and 50")
	}

	charPool := ""
	if input.IncludeUppercase {
		charPool += uppercaseChars
	}
	if input.IncludeLowercase {
		charPool += lowercaseChars
	}
	if input.IncludeNumbers {
		charPool += numberChars
	}
	if input.IncludeSymbols {
		charPool += symbolChars
	}

	if charPool == "" {
		return errResult("at least one character set must be selected (uppercase, lowercase, numbers, or symbols)")
	}

	poolLen := len(charPool)
	entropy := math.Log2(math.Pow(float64(poolLen), float64(length)))
	entropy = math.Round(entropy*100) / 100

	password := make([]byte, length)
	if _, err := rand.Read(password); err != nil {
		return errResult("failed to generate random password: " + err.Error())
	}

	for i := range password {
		password[i] = charPool[int(password[i])%poolLen]
	}

	return resultJSON(PasswordGenerateOutput{
		Password: string(password),
		Length:   length,
		Entropy:  entropy,
	})
}
