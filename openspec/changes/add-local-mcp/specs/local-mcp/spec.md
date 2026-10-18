## Purpose

Make the existing Advocate tool suite available without Kafka through an
explicitly temporary local session.

## ADDED Requirements

### Requirement: Explicit local storage selection
`trial mcp --local` SHALL create one fresh in-memory session log without opening
a broker connection. It SHALL ignore ambient `TRIAL_BROKER` and reject an
explicit `--broker` option while local mode is active. Default mode SHALL retain
its existing eager Kafka connection behavior.

#### Scenario: Local mode with an unusable environment broker
- **WHEN** the client starts `trial mcp --local` with an unusable `TRIAL_BROKER`
- **THEN** initialization, discovery, and tool calls work without connecting to that broker

#### Scenario: Conflicting explicit options
- **WHEN** `--local` is active and `--broker` is supplied explicitly
- **THEN** the command returns argument status 2 before starting the server

### Requirement: Session lifetime is visible and isolated
Local initialization instructions SHALL identify temporary state and explain
that case identifiers cannot resume work after the process exits. The local
server SHALL expose the existing tools and result formats. Each process SHALL
start with empty session storage. `trial_test` SHALL remain isolated from that
session storage.

#### Scenario: Case workflow in one session
- **WHEN** a client files, serves, proceeds, and observes a case in local mode
- **THEN** the existing tools operate on that session's case and output

#### Scenario: New process after exit
- **WHEN** a local session ends and a new local process starts
- **THEN** the new process contains none of the previous cases or statutes

#### Scenario: Inline statute test
- **WHEN** a local client tests a program with inline enactments
- **THEN** the test runs without adding those statutes or cases to session storage

### Requirement: Retain protocol shutdown behavior
Local mode SHALL exit successfully after complete input reaches EOF. Unix
SIGINT and SIGTERM SHALL cancel idle or partial-line input through the existing
CLI cancellation mechanism. Existing protocol and tool resource bounds SHALL
continue to apply.

#### Scenario: EOF after requests
- **WHEN** the client finishes complete requests and closes standard input
- **THEN** the server finishes its responses and exits with status 0

#### Scenario: Signal during partial input
- **WHEN** SIGINT or SIGTERM arrives while the client holds an incomplete request
- **THEN** the local server exits with status 1 without waiting for the rest of the line
