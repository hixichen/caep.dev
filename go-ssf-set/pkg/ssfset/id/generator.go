package id

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// Generator defines the interface for ID generation
type Generator interface {
	Generate() string
}

// UUIDGenerator generates UUIDs for SET IDs
type UUIDGenerator struct{}

// NewUUIDGenerator creates a new UUID-based generator
func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

// Generate creates a new UUID string
func (g *UUIDGenerator) Generate() string {
	return uuid.New().String()
}

// SequentialGenerator generates sequential IDs with a prefix
type SequentialGenerator struct {
	prefix    string
	sequence  uint64
	padLength int
}

// NewSequentialGenerator creates a new sequential generator
func NewSequentialGenerator(prefix string, padLength int) *SequentialGenerator {
	return &SequentialGenerator{
		prefix:    prefix,
		sequence:  0,
		padLength: padLength,
	}
}

// Generate creates a new sequential ID
func (g *SequentialGenerator) Generate() string {
	seq := atomic.AddUint64(&g.sequence, 1)

	return fmt.Sprintf("%s%0*d", g.prefix, g.padLength, seq)
}

// TimestampGenerator generates timestamp-based IDs
type TimestampGenerator struct {
	prefix string
	suffix Generator // Optional suffix generator
}

// NewTimestampGenerator creates a new timestamp-based generator
func NewTimestampGenerator(prefix string) *TimestampGenerator {
	return &TimestampGenerator{
		prefix: prefix,
	}
}

// WithSuffix adds a suffix generator
func (g *TimestampGenerator) WithSuffix(suffix Generator) *TimestampGenerator {
	g.suffix = suffix

	return g
}

// Generate creates a new timestamp-based ID
func (g *TimestampGenerator) Generate() string {
	timestamp := time.Now().UnixNano()
	if g.suffix != nil {
		return fmt.Sprintf("%s%d%s", g.prefix, timestamp, g.suffix.Generate())
	}

	return fmt.Sprintf("%s%d", g.prefix, timestamp)
}

// RandomGenerator generates random IDs
type RandomGenerator struct {
	length   int
	encoding Encoding
}

// Encoding represents the encoding for random IDs
type Encoding int

const (
	EncodingHex Encoding = iota
	EncodingBase64
	EncodingBase64URL
)

// NewRandomGenerator creates a new random generator
func NewRandomGenerator(length int, encoding Encoding) *RandomGenerator {
	return &RandomGenerator{
		length:   length,
		encoding: encoding,
	}
}

// Generate creates a new random ID
func (g *RandomGenerator) Generate() string {
	// Calculate number of random bytes needed
	byteLength := g.length
	switch g.encoding {
	case EncodingBase64, EncodingBase64URL:
		byteLength = (g.length * 3) / 4
	case EncodingHex:
		byteLength = g.length / 2
	}

	// Generate random bytes
	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to UUID in case of error
		return uuid.New().String()
	}

	// Encode according to specified encoding
	switch g.encoding {
	case EncodingBase64:
		return base64.StdEncoding.EncodeToString(bytes)[:g.length]
	case EncodingBase64URL:
		return base64.URLEncoding.EncodeToString(bytes)[:g.length]
	default: // EncodingHex
		return hex.EncodeToString(bytes)[:g.length]
	}
}

// CustomGenerator allows for custom ID generation logic
type CustomGenerator struct {
	generateFn func() string
}

// NewCustomGenerator creates a new generator with custom logic
func NewCustomGenerator(generateFn func() string) *CustomGenerator {
	return &CustomGenerator{
		generateFn: generateFn,
	}
}

// Generate creates a new ID using the custom function
func (g *CustomGenerator) Generate() string {
	return g.generateFn()
}

// Common generators
var (
	// UUID generates standard UUIDs
	UUID = NewUUIDGenerator()

	// Random32 generates 32-character random hex strings
	Random32 = NewRandomGenerator(32, EncodingHex)

	// Base64URL20 generates 20-character base64url-encoded random strings
	Base64URL20 = NewRandomGenerator(20, EncodingBase64URL)

	// TimestampHex generates timestamp-based IDs with random hex suffixes
	TimestampHex = NewTimestampGenerator("").WithSuffix(
		NewRandomGenerator(8, EncodingHex),
	)
)

// Predefined prefixes for sequential generators
const (
	PrefixSET = "set_"
	PrefixTXN = "txn_"
)

// Additional helper generators
var (
	// Sequential generates IDs like "set_0001"
	Sequential = NewSequentialGenerator(PrefixSET, 4)

	// TransactionSequential generates IDs like "txn_0001"
	TransactionSequential = NewSequentialGenerator(PrefixTXN, 4)
)

// ValidateID checks if an ID matches the expected format
func ValidateID(id string, minLength, maxLength int) error {
	if len(id) < minLength {
		return fmt.Errorf("ID too short: minimum length is %d", minLength)
	}

	if maxLength > 0 && len(id) > maxLength {
		return fmt.Errorf("ID too long: maximum length is %d", maxLength)
	}

	return nil
}
