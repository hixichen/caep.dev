# caep.dev Libraries

## [secevent](./secevent)
A comprehensive Go library for building, signing, parsing, and validating Security Event Tokens (SecEvents) according to the Security Event Token [RFC 8417](https://tools.ietf.org/html/rfc8417).

## [ssfreceiver](./ssfreceiver)
A Go library for implementing [Shared Signals Framework (SSF)](https://openid.github.io/sharedsignals/openid-sharedsignals-framework-1_0.html) receivers.

## [ssf-hub](./ssf-hub)
A centralized SSF (Shared Signals Framework) hub service that acts as an event distribution broker using Google Cloud Pub/Sub as the backend. Provides standards-compliant SSF receiver endpoints, event brokering capabilities, and a registration API for managing multiple receivers.

### [archive](./archive)
Contains the original SSF receiver implementation. This has been superseded by the new ssfreceiver library and is maintained for historical reference only.

## Contributing

Contributions to the project are welcome, including feature enhancements, bug fixes, and documentation improvements.