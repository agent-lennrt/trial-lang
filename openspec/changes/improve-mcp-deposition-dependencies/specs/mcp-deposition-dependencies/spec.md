## Purpose

Allow MCP clients to test imported offices using explicitly supplied statutes
and the existing isolated deposition runner.

## ADDED Requirements

### Requirement: Resolve inline enactments by exact label
`trial_test` SHALL accept an optional `enactments` object mapping exact deposition
`ENACT` labels to string sources. It SHALL resolve those sources in declaration
order, including repeated labels, without reading files or the server log.

#### Scenario: Transitive imports
- **WHEN** the caller supplies a dependency and an importing statute with their labels declared in dependency order
- **THEN** the isolated deposition can call the imported office and verify its result

#### Scenario: Repeated label
- **WHEN** a deposition declares the same `ENACT` label twice
- **THEN** the supplied source is enacted at both positions in the sequence

#### Scenario: Existing test has no dependencies
- **WHEN** a deposition has no `ENACT` statements and omits `enactments`
- **THEN** the tool retains its existing execution behavior and result fields

### Requirement: Validate the complete dependency mapping
The tool SHALL reject missing or unused keys and null or non-string sources as
tool argument errors before execution. It SHALL allow at most 100 map entries,
4 MiB per statute, and 256 MiB of expanded statute source counting repeated
declarations. The existing 16 MiB JSON request limit SHALL continue to apply.

#### Scenario: Missing or unused source
- **WHEN** a declared label has no source or a supplied source has no declaration
- **THEN** the tool reports a failed-tool result before running the deposition

#### Scenario: Malformed or excessive source
- **WHEN** an inline value is null, has a non-string type, or exceeds a count or byte bound
- **THEN** the tool rejects the input without accessing persistent storage

#### Scenario: Invalid statute source
- **WHEN** a complete bounded mapping contains an invalid statute source
- **THEN** the ordinary deposition result reports a contradiction with `consistent: false`
