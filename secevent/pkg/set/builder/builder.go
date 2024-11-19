// pkg/set/builder/builder.go
package builder

import (
	"github.com/sgnl-ai/caep.dev-receiver/secevent/pkg/set/id"
)

// Builder provides configuration for creating SETs
type Builder struct {
	defaultIssuer      string
	defaultIDGenerator id.Generator
}

// Option defines the function signature for builder options
type Option func(*Builder)

// WithDefaultIssuer sets the default issuer for all SETs created by this builder
func WithDefaultIssuer(issuer string) Option {
	return func(b *Builder) {
		b.defaultIssuer = issuer
	}
}

// WithDefaultIDGenerator sets the default ID generator for all SETs created by this builder
func WithDefaultIDGenerator(generator id.Generator) Option {
	return func(b *Builder) {
		b.defaultIDGenerator = generator
	}
}

// NewBuilder creates a new SET builder with the provided options
func NewBuilder(opts ...Option) *Builder {
	b := &Builder{
		defaultIDGenerator: id.NewUUIDGenerator(), // Default to UUID generator
	}

	// Apply options
	for _, opt := range opts {
		opt(b)
	}

	return b
}

// NewSET creates a new multi-event SET with default configurations
func (b *Builder) NewSET() *SET {
	set := newSET()
	if b.defaultIssuer != "" {
		set.WithIssuer(b.defaultIssuer)
	}

	if b.defaultIDGenerator != nil {
		set.WithID(b.defaultIDGenerator.Generate())
	}

	return set
}

// NewSingleEventSET creates a new single-event SET with default configurations
func (b *Builder) NewSingleEventSET() *SingleEventSET {
	set := newSingleEventSET()
	if b.defaultIssuer != "" {
		set.WithIssuer(b.defaultIssuer)
	}

	if b.defaultIDGenerator != nil {
		set.WithID(b.defaultIDGenerator.Generate())
	}

	return set
}
