package builder

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/event"
	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/id"
	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/subject"
)

// Signer defines the interface for signing tokens
type Signer interface {
	Sign(token *jwt.Token) (string, error)
}

// DefaultSigner uses the standard JWT signing methods
type DefaultSigner struct {
	signingKey    crypto.PrivateKey
	signingMethod jwt.SigningMethod
}

func NewDefaultSigner(signingKey crypto.PrivateKey, signingMethod jwt.SigningMethod) *DefaultSigner {
	return &DefaultSigner{
		signingKey:    signingKey,
		signingMethod: signingMethod,
	}
}

func (s *DefaultSigner) Sign(token *jwt.Token) (string, error) {
	token.Method = s.signingMethod

	return token.SignedString(s.signingKey)
}

// SETBuilder builds Security Event Tokens
type SETBuilder struct {
	issuer      string
	idGenerator id.Generator
	signer      Signer
	keyID       *string
}

// NewSETBuilder creates a new SET builder
func NewSETBuilder() *SETBuilder {
	return &SETBuilder{
		idGenerator: id.NewUUIDGenerator(),
	}
}

// WithSigner sets the custom signer
func (b *SETBuilder) WithSigner(signer Signer) *SETBuilder {
	b.signer = signer

	return b
}

// WithSigningKey sets the signing key and method
func (b *SETBuilder) WithSigningKey(key crypto.PrivateKey) *SETBuilder {
	var signingMethod jwt.SigningMethod

	// Determine the signing method based on key type
	switch key.(type) {
	case *rsa.PrivateKey:
		signingMethod = jwt.SigningMethodRS256
	case *ecdsa.PrivateKey:
		signingMethod = jwt.SigningMethodES256
	default:
		panic(fmt.Sprintf("unsupported key type: %T", key))
	}

	b.signer = NewDefaultSigner(key, signingMethod)

	return b
}

// WithSigningMethod sets a custom signing method
func (b *SETBuilder) WithSigningMethod(method jwt.SigningMethod) *SETBuilder {
	if defaultSigner, ok := b.signer.(*DefaultSigner); ok {
		defaultSigner.signingMethod = method
	} else {
		panic("Cannot set signing method when custom signer is used")
	}

	return b
}

// WithKeyID sets the key ID to be included in the JWT header
func (b *SETBuilder) WithKeyID(kid string) *SETBuilder {
	b.keyID = &kid

	return b
}

// WithIssuer sets the issuer for all SETs created by this builder
func (b *SETBuilder) WithIssuer(issuer string) *SETBuilder {
	b.issuer = issuer

	return b
}

// WithIDGenerator sets the ID generator for created SETs
func (b *SETBuilder) WithIDGenerator(generator id.Generator) *SETBuilder {
	b.idGenerator = generator

	return b
}

// NewSet creates a new SET with the builder's configuration
func (b *SETBuilder) NewSet() *SETCreator {
	set := NewSET()
	set.WithIssuer(b.issuer)
	set.WithIssuedAt(time.Now().Unix())

	if b.idGenerator != nil {
		set.WithID(b.idGenerator.Generate())
	}

	return &SETCreator{
		set:    set,
		signer: b.signer,
		keyID:  b.keyID,
	}
}

// Validate checks if the builder is properly configured
func (b *SETBuilder) Validate() error {
	if b.issuer == "" {
		return fmt.Errorf("issuer is required")
	}

	return nil
}

// SETCreator creates individual SETs using the builder's configuration
type SETCreator struct {
	set    *SET
	signer Signer
	keyID  *string
}

// WithSubject sets the subject for this SET
func (c *SETCreator) WithSubject(sub subject.Subject) *SETCreator {
	c.set.WithSubject(sub)

	return c
}

// WithEvent adds an event to this SET
func (c *SETCreator) WithEvent(evt event.Event) *SETCreator {
	c.set.WithEvent(evt)

	return c
}

// WithAudience sets the audience for this SET
func (c *SETCreator) WithAudience(audience ...string) *SETCreator {
	c.set.WithAudience(audience...)

	return c
}

// WithTransactionID sets the transaction ID for this SET
func (c *SETCreator) WithTransactionID(txn string) *SETCreator {
	c.set.WithTransactionID(txn)

	return c
}

// BuildSigned creates a signed JWT containing the SET
func (c *SETCreator) BuildSigned() (string, error) {
	if c.signer == nil {
		return "", fmt.Errorf("signer is required for signed SETs")
	}

	if err := c.set.Validate(); err != nil {
		return "", fmt.Errorf("invalid SET: %w", err)
	}

	// Create JWT with SET as claims
	token := jwt.NewWithClaims(jwt.SigningMethodNone, c.set)

	// Set JWT headers
	token.Header["typ"] = "secevent+jwt"
	if c.keyID != nil {
		token.Header["kid"] = *c.keyID
	}

	// Use the signer to sign the token
	signedString, err := c.signer.Sign(token)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedString, nil
}

// BuildUnsigned creates an unsigned JWT containing the SET
func (c *SETCreator) BuildUnsigned() (string, error) {
	if err := c.set.Validate(); err != nil {
		return "", fmt.Errorf("invalid SET: %w", err)
	}

	// Create JWT with SET as claims and 'none' algorithm
	token := jwt.NewWithClaims(jwt.SigningMethodNone, c.set)

	token.Header["typ"] = "secevent+jwt"

	// Get the signing string (header and payload)
	signingString, err := token.SigningString()
	if err != nil {
		return "", err
	}

	// Append the empty signature part
	return signingString + ".", nil
}
