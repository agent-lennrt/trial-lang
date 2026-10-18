## ADDED Requirements

### Requirement: Compile source without execution

The CLI SHALL validate Form K-1 and Form S-1 source using an isolated in-memory
log and the existing filing rules. It MUST NOT execute case instructions.

#### Scenario: Source would wait or fail at runtime
- **WHEN** accepted source contains an input wait, an infinite article loop, or a runtime-only error
- **THEN** check succeeds without waiting, executing, or printing proclamations

#### Scenario: Invalid filing
- **WHEN** source fails parsing, compilation, or incorporation
- **THEN** check returns status 1 and a diagnostic on stderr

#### Scenario: Supplemental form
- **WHEN** standalone Form K-2 source is checked
- **THEN** check returns status 1 explaining that an existing case is required

### Requirement: Resolve bounded local dependencies

The CLI SHALL accept `--canon` and ordered repeated `--enact` paths, resolving
dependencies on temporary storage before checking the target source. It SHALL
apply the local-run source bounds and accept standard input for at most one source.

#### Scenario: Canon and custom imports
- **WHEN** all dependencies are supplied in dependency order
- **THEN** check resolves them without Kafka and discards the temporary records

#### Scenario: Invalid arguments or oversized input
- **WHEN** flags are invalid or more than one source uses standard input
- **THEN** check returns status 2 before reading source

#### Scenario: Source exceeds its byte bound
- **WHEN** an input source exceeds 4 MiB
- **THEN** check returns status 1 without accepting the source

### Requirement: Bounded completion and output

The CLI SHALL apply a positive timeout defaulting to 30 seconds and the existing
one-second process cancellation grace. A successful check SHALL print only
`Source accepted.` and a newline to stdout. Cancellation and output failures
SHALL return status 1.

#### Scenario: Stdin remains open
- **WHEN** standard input does not finish before the deadline
- **THEN** the process returns status 1 within the cancellation grace

#### Scenario: Successful check
- **WHEN** validation succeeds before the deadline
- **THEN** the command prints the success line and returns status 0
