package parser

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/builder"

	_ "github.com/sgnl-ai/go-ssf-set/pkg/ssfset/event/caep" // Initialize CAEP events
	_ "github.com/sgnl-ai/go-ssf-set/pkg/ssfset/event/ssf"  // Initialize SSF events
)

// Parser parses and validates Security Event Tokens
type Parser struct {
	keyFunc          jwt.Keyfunc
	expectedIssuer   string
	expectedAudience string

	jwksURL   string
	jwksJSON  []byte
	publicKey interface{}

	jwksKeys map[string]interface{}
}

// NewParser creates a new SET parser
func NewParser() *Parser {
	return &Parser{}
}

// WithJWKSURL sets a JWKS URL to fetch keys from
func (p *Parser) WithJWKSURL(url string) *Parser {
	p.jwksURL = url
	p.keyFunc = p.jwksKeyFunc

	p.fetchJWKS()

	return p
}

// WithJWKSJSON sets the JWKS JSON data
func (p *Parser) WithJWKSJSON(jwksJSON []byte) *Parser {
	p.jwksJSON = jwksJSON
	p.keyFunc = p.jwksKeyFunc

	p.parseJWKS()

	return p
}

// WithPublicKey sets a direct public key
func (p *Parser) WithPublicKey(key interface{}) *Parser {
	p.publicKey = key
	p.keyFunc = func(token *jwt.Token) (interface{}, error) {
		return key, nil
	}

	return p
}

// WithExpectedIssuer sets the expected issuer for validation
func (p *Parser) WithExpectedIssuer(issuer string) *Parser {
	p.expectedIssuer = issuer

	return p
}

// WithExpectedAudience sets the expected audience for validation
func (p *Parser) WithExpectedAudience(audience string) *Parser {
	p.expectedAudience = audience

	return p
}

// fetchJWKS retrieves the JWKS from the provided URL
func (p *Parser) fetchJWKS() error {
	resp, err := http.Get(p.jwksURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS from URL: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch JWKS from URL: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read JWKS response: %w", err)
	}

	p.jwksJSON = body

	return p.parseJWKS()
}

// parseJWKS parses the JWKS JSON and stores the keys
func (p *Parser) parseJWKS() error {
	var jwks struct {
		Keys []json.RawMessage `json:"keys"`
	}

	if err := json.Unmarshal(p.jwksJSON, &jwks); err != nil {
		return fmt.Errorf("failed to parse JWKS: %w", err)
	}

	p.jwksKeys = make(map[string]interface{})
	for _, keyData := range jwks.Keys {
		var key map[string]interface{}
		if err := json.Unmarshal(keyData, &key); err != nil {
			return fmt.Errorf("failed to parse JWKS key: %w", err)
		}

		kid, ok := key["kid"].(string)
		if !ok || kid == "" {
			continue // Skip keys without kid
		}

		kty, ok := key["kty"].(string)
		if !ok {
			continue // Skip keys without kty
		}

		// Parse the key based on kty
		var parsedKey interface{}
		var err error

		switch kty {
		case "RSA":
			parsedKey, err = parseRSAPublicKey(key)
		case "EC":
			parsedKey, err = parseECPublicKey(key)
		default:
			continue // Unsupported key type
		}

		if err != nil {
			return fmt.Errorf("failed to parse key with kid %s: %w", kid, err)
		}

		p.jwksKeys[kid] = parsedKey
	}

	return nil
}

// parseRSAPublicKey parses RSA public key from JWKS key data
func parseRSAPublicKey(key map[string]interface{}) (*rsa.PublicKey, error) {
	nStr, ok := key["n"].(string)
	if !ok {
		return nil, fmt.Errorf("missing modulus 'n' in RSA key")
	}

	eStr, ok := key["e"].(string)
	if !ok {
		return nil, fmt.Errorf("missing exponent 'e' in RSA key")
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus 'n': %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent 'e': %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)

	var e int
	if len(eBytes) == 3 {
		e = int(eBytes[0])<<16 | int(eBytes[1])<<8 | int(eBytes[2])
	} else if len(eBytes) == 1 {
		e = int(eBytes[0])
	} else {
		return nil, fmt.Errorf("unexpected exponent size")
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

// parseECPublicKey parses EC public key from JWKS key data
func parseECPublicKey(key map[string]interface{}) (*ecdsa.PublicKey, error) {
	crv, ok := key["crv"].(string)
	if !ok {
		return nil, fmt.Errorf("missing curve 'crv' in EC key")
	}

	xStr, ok := key["x"].(string)
	if !ok {
		return nil, fmt.Errorf("missing x coordinate in EC key")
	}

	yStr, ok := key["y"].(string)
	if !ok {
		return nil, fmt.Errorf("missing y coordinate in EC key")
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(xStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode x coordinate: %w", err)
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(yStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode y coordinate: %w", err)
	}

	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)

	var curve elliptic.Curve
	switch crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported curve: %s", crv)
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}, nil
}

// jwksKeyFunc retrieves the key from JWKS based on the kid
func (p *Parser) jwksKeyFunc(token *jwt.Token) (interface{}, error) {
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("token header does not contain 'kid'")
	}

	key, ok := p.jwksKeys[kid]
	if !ok {
		return nil, fmt.Errorf("no key found for kid %s", kid)
	}

	return key, nil
}

// ParseSetVerify parses and verifies a signed SET
func (p *Parser) ParseSetVerify(tokenString string) (*builder.SET, error) {
	var set builder.SET

	// Ensure keyFunc is set
	if p.keyFunc == nil {
		return nil, fmt.Errorf("no key function is set for signature verification")
	}

	// Create a new Parser with validation options
	parser := jwt.NewParser(
		jwt.WithIssuer(p.expectedIssuer),
		jwt.WithAudience(p.expectedAudience),
		jwt.WithValidMethods([]string{"RS256", "ES256"}), // specify acceptable methods
	)

	// Parse the token with claims
	token, err := parser.ParseWithClaims(tokenString, &set, p.keyFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Check if the token is valid
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return &set, nil
}

// ParseSetNoVerify parses a SET without signature verification
func (p *Parser) ParseSetNoVerify(tokenString string) (*builder.SET, error) {
	var set builder.SET

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())

	_, _, err := parser.ParseUnverified(tokenString, &set)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return &set, nil
}
