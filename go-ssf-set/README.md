# Go SSF SET Library

A comprehensive Go library for building, signing, parsing, and validating Security Event Tokens (SETs) according to the Shared Signals Framework (SSF) specification.

---

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
  - [Building a SET](#building-a-set)
  - [Parsing a SET](#parsing-a-set)
- [Defining Custom Events](#defining-custom-events)
- [Using Custom Signers](#using-custom-signers)
- [Subjects and Identifiers](#subjects-and-identifiers)
- [ID Generators](#id-generators)
- [Event Handling](#event-handling)
- [Providing Verification Keys](#providing-verification-keys)
- [Contributing](#contributing)

---

## Features

- **Complete SSF SET Specification Implementation**: Supports building, signing, parsing, and validating Security Event Tokens as per the SSF specification.
- **Standard and Custom Events**: Out-of-the-box support for standard CAEP and SSF events, with the ability to define and handle custom events.
- **Flexible Subject Identifiers**: Supports various subject identifier formats, including email, phone number, issuer and subject pairs, URIs, and more.
- **Extensible Signing Mechanisms**: Integrate with custom signing functions or hardware security modules (HSMs) when private keys are not directly accessible.
- **Flexible Key Provisioning for Parsing**: Supports multiple ways to provide verification keys, such as JWKS URLs, JWKS JSON, or direct public keys.
- **Rich Examples**: Comprehensive examples demonstrating how to use the library for common and advanced use cases.

---

## Installation

```bash
go get github.com/sgnl-ai/caep.dev/go-ssf-set
```

---

## Quick Start

### Building a SET

```go
package main

import (
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
    "fmt"
    "time"

    "github.com/sgnl-ai/caep.dev/go-ssf-set/pkg/ssfset/builder"
    "github.com/sgnl-ai/caep.dev/go-ssf-set/pkg/ssfset/event/caep"
    "github.com/sgnl-ai/caep.dev/go-ssf-set/pkg/ssfset/id"
    "github.com/sgnl-ai/caep.dev/go-ssf-set/pkg/ssfset/subject"
)

func main() {
    // Generate a private key (for example purposes)
    privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        panic(err)
    }

    // Create SET builder with default signer
    setBuilder := builder.NewSETBuilder().
        WithSigningKey(privateKey).
        WithKeyID("key-1").
        WithIssuer("https://issuer.example.com").
        WithIDGenerator(id.UUID) // Use UUID generator for SET IDs

    // Create a session revoked event
    sessionEvent := caep.NewSessionRevokedEvent().
        WithInitiatingEntity(caep.InitiatingEntityPolicy).
        WithReasonAdmin("en", "Security policy violation").
        WithEventTimestamp(time.Now().Unix())

    // Create a subject (e.g., email)
    userEmail, err := subject.NewEmailSubject("user@example.com")
    if err != nil {
        panic(err)
    }

    // Build signed SET
    signedSet, err := setBuilder.NewSet().
        WithSubject(userEmail).
        WithEvent(sessionEvent).
        WithAudience("https://receiver.example.com").
        BuildSigned()
    if err != nil {
        panic(err)
    }

    fmt.Println("Signed SET:", signedSet)

    // Build unsigned SET
    unsignedSet, err := setBuilder.NewSet().
        WithSubject(userEmail).
        WithEvent(sessionEvent).
        WithAudience("https://receiver.example.com").
        BuildUnsigned()
    if err != nil {
        panic(err)
    }

    fmt.Println("Unsigned SET:", unsignedSet)
}
```

### Parsing a SET

```go
package main

import (
	"fmt"

	"github.com/sgnl-ai/caep.dev/go-ssf-set/pkg/ssfset/parser"
)

func main() {
	tokenString := "..." // Your SET JWT string

	// Create a parser with JWKS URL
	setParser := parser.NewParser().
		WithJWKSURL("https://issuer.example.com/jwks.json").
		WithExpectedIssuer("https://issuer.example.com").
		WithExpectedAudience("https://receiver.example.com")

	// Parse SET with signature verification
	verifiedParsedSet, err := setParser.ParseSetVerify(tokenString)
	if err != nil {
		panic(err)
	}

	// Access SET's events
	event := verifiedParsedSet.Event
	fmt.Printf("Event Type: %s\n", event.Type())

	// Access SET's subject
	subject := verifiedParsedSet.Subject
	fmt.Printf("Subject Format: %s\n", subject.Format())
}
```

---

## Defining Custom Events

The library allows you to define custom events.

**Example: Defining and Using a Custom Event**

```go
package main

import (
    "encoding/json"
    "fmt"

    "github.com/sgnl-ai/caep.dev/go-ssf-set/pkg/ssfset/builder"
    "github.com/sgnl-ai/caep.dev/go-ssf-set/pkg/ssfset/event"
    "github.com/sgnl-ai/caep.dev/go-ssf-set/pkg/ssfset/id"
    "github.com/sgnl-ai/caep.dev/go-ssf-set/pkg/ssfset/subject"
)

// Define your custom event type
const CustomEventType event.EventType = "https://example.com/event-type/custom"

// CustomEvent represents a custom event
type CustomEvent struct {
    event.BaseEvent
    CustomField string `json:"custom_field"`
}

// NewCustomEvent creates a new custom event
func NewCustomEvent(customField string) *CustomEvent {
    e := &CustomEvent{
        CustomField: customField,
    }
    e.SetType(CustomEventType)
    return e
}

// Validate ensures the event is valid
func (e *CustomEvent) Validate() error {
    if e.CustomField == "" {
        return event.NewError(event.ErrCodeMissingValue, "custom_field is required", "custom_field")
    }
    return nil
}

// MarshalJSON implements the json.Marshaler interface
func (e *CustomEvent) MarshalJSON() ([]byte, error) {
    payload := map[string]interface{}{
        "custom_field": e.CustomField,
    }
    wrapper := struct {
        Events map[event.EventType]interface{} `json:"events"`
    }{
        Events: map[event.EventType]interface{}{
            e.Type(): payload,
        },
    }
    return json.Marshal(wrapper)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (e *CustomEvent) UnmarshalJSON(data []byte) error {
    var wrapper struct {
        Events map[event.EventType]json.RawMessage `json:"events"`
    }

    if err := json.Unmarshal(data, &wrapper); err != nil {
        return err
    }

    eventData, ok := wrapper.Events[e.Type()]
    if !ok {
        return event.NewError(event.ErrCodeInvalidEventType, "event type not found", "events")
    }

    var payload struct {
        CustomField string `json:"custom_field"`
    }

    if err := json.Unmarshal(eventData, &payload); err != nil {
        return err
    }

    e.CustomField = payload.CustomField
    return e.Validate()
}

// Register the custom event parser
func init() {
    event.RegisterEventParser(CustomEventType, func(data json.RawMessage) (event.Event, error) {
        var e CustomEvent
        if err := json.Unmarshal(data, &e); err != nil {
            return nil, err
        }
        return &e, nil
    })
}

func main() {
    // Create a custom event
    customEvent := NewCustomEvent("Custom Value")

    // Create a subject
    userEmail, err := subject.NewEmailSubject("user@example.com")
    if err != nil {
        panic(err)
    }

    // Build the SET
    setBuilder := builder.NewSETBuilder().
        WithIssuer("https://issuer.example.com").
        WithIDGenerator(id.UUID)

    // Build unsigned SET with custom event
    unsignedSet, err := setBuilder.NewSet().
        WithSubject(userEmail).
        WithEvent(customEvent).
        BuildUnsigned()
    if err != nil {
        panic(err)
    }

    fmt.Println("Unsigned SET with Custom Event:", unsignedSet)
}
```

---

## Using Custom Signers

The library supports custom signing mechanisms, allowing integration with HSMs or external signing services where private keys are not directly accessible.

**Implementing a Custom Signer**

```go
package main

import (
    "strings"

    "github.com/golang-jwt/jwt/v5"
    "github.com/sgnl-ai/caep.dev/go-ssf-set/pkg/ssfset/builder"
)

// CustomSigner implements the Signer interface
type CustomSigner struct {
    // Fields for HSM client or external service configuration
}

func (s *CustomSigner) Sign(token *jwt.Token) (string, error) {
    // Obtain the signing string
    signingString, err := token.SigningString()
    if err != nil {
        return "", err
    }

    // Use your HSM or external service to sign the string
    signature, err := externalSign(signingString)
    if err != nil {
        return "", err
    }

    // Return the complete JWT
    return strings.Join([]string{signingString, signature}, "."), nil
}

// externalSign is a placeholder function representing the external signing process
func externalSign(signingString string) (string, error) {
    // Implement the signing logic using your HSM or external service
    // Return the signature encoded appropriately (e.g., base64url)
    return "signature", nil
}

func main() {
    customSigner := &CustomSigner{}

    setBuilder := builder.NewSETBuilder().
        WithSigner(customSigner).
        WithKeyID("key-1").
        WithIssuer("https://issuer.example.com")

    // Build and sign the SET as usual
}
```

---

## Subjects and Identifiers

The library supports various subject identifier formats as per the SSF specification.

**Supported Formats**

- `email`: Identified by an email address.
- `phone_number`: Identified by a phone number.
- `iss_sub`: Identified by an issuer and subject pair.
- `uri`: Identified by a URI.
- `opaque`: Identified by an opaque identifier.
- `account`: Identified by an `acct:` URI.
- `did`: Identified by a Decentralized Identifier (DID) URL.
- `jwt_id`: Identified by a JWT issuer and ID.
- `saml_assertion_id`: Identified by a SAML assertion issuer and ID.
- `complex`: A composite subject made up of multiple components.

**Example: Creating a Complex Subject**

```go
userEmail, _ := subject.NewEmailSubject("user@example.com")
deviceID, _ := subject.NewOpaqueSubject("device-12345")

complexSubject := subject.NewComplexSubject().
    WithUser(userEmail).
    WithDevice(deviceID)
```

---

## ID Generators

Customize how SET IDs (`jti`) are generated.

**Built-in Generators**

- `id.UUID`: Generates UUIDs.
- `id.Random32`: Generates 32-character random hex strings.
- `id.Base64URL20`: Generates 20-character base64url-encoded random strings.
- `id.Sequential`: Generates sequential IDs with a prefix.

**Example: Using a Custom ID Generator**

```go
customGenerator := id.NewCustomGenerator(func() string {
    return "custom-id-12345"
})

setBuilder := builder.NewSETBuilder().
    WithIDGenerator(customGenerator)
```

---

## Event Handling

The library supports both standard events and custom events.

**Standard Events**

- **CAEP Events**: `session-revoked`, `credential-change`, `assurance-level-change`, `token-claim-change`, and `device-compliance-change`.
- **SSF Events**: `verification` and `stream-updated`.

**Custom Events**

Define your own event types by implementing the `event.Event` interface and registering a parser.

**Registering a Custom Event Parser**

```go
func init() {
    event.RegisterEventParser(CustomEventType, parseCustomEvent)
}
```

---

## Providing Verification Keys

When parsing and verifying SETs, you can provide verification keys in multiple ways:

**Using JWKS URL**

```go
setParser := parser.NewParser().
    WithJWKSURL("https://issuer.example.com/jwks.json").
    WithExpectedIssuer("https://issuer.example.com").
    WithExpectedAudience("https://receiver.example.com")
```

**Using JWKS JSON**

```go
jwksJSON := []byte(`{"keys": [...]}`) // Replace with actual JWKS JSON
setParser := parser.NewParser().
    WithJWKSJSON(jwksJSON).
    WithExpectedIssuer("https://issuer.example.com").
    WithExpectedAudience("https://receiver.example.com")
```

**Using Direct Public Key**

```go
publicKey := /* Load your public key */
setParser := parser.NewParser().
    WithPublicKey(publicKey).
    WithExpectedIssuer("https://issuer.example.com").
    WithExpectedAudience("https://receiver.example.com")
```

---

## Contributing

Contributions to the project are welcome, including feature enhancements, bug fixes, and documentation improvements.