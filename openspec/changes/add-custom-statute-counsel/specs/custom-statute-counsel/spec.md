## Purpose

Validate explicitly configured custom statutes while editing and locate each
rejection in the source that owns it.

## ADDED Requirements

### Requirement: Explicit bounded dependency context
Counsel SHALL accept repeated `--enact <path>` inputs in dependency order, after
optional bundled canon. It MUST reject standard-input paths and duplicate
normalized paths. Inputs MUST fit 100 files, 4 MiB each, and 256 MiB aggregate.
Only configured files SHALL supply custom imports. Explicit sources SHALL make
unknown imports diagnostic errors. Default and canon-only fallback SHALL remain.

#### Scenario: Custom and bundled dependencies
- **WHEN** a document uses offices from configured statutes and enabled canon
- **THEN** Counsel checks the real incorporation and compilation rules without executing it

#### Scenario: Invalid or absent configured source
- **WHEN** an initial configured file cannot be read within its bounds
- **THEN** startup fails with a useful error
- **WHEN** its text is not a valid statute
- **THEN** the server reports the invalid text through editor diagnostics

#### Scenario: Unconfigured open statute
- **WHEN** an open document names a statute that was not configured
- **THEN** it does not satisfy another document's missing import

### Requirement: Accurate diagnostic ownership
Parser, examiner, compiler, and incorporation failures SHALL retain owning source
identity. Counsel MUST use that source's URI and UTF-16 range. Bundled locations
SHALL use stable virtual URIs with a contextual diagnostic in the importing
document for clients that cannot open virtual sources. Related information SHALL
identify importing
locations when available. Ownership metadata MUST NOT change error strings,
instruction positions, serialized records, or the public canon API.

#### Scenario: Imported declaration fails
- **WHEN** compilation rejects a nested or deferred statement in a transitive dependency
- **THEN** the diagnostic points to that dependency's source and line

#### Scenario: Duplicate declaration
- **WHEN** an imported declaration conflicts with an earlier declaration
- **THEN** the rejection belongs to the declaration rejected by the compiler

#### Scenario: Unicode and line endings
- **WHEN** a rejection follows supplementary Unicode or uses CRLF source lines
- **THEN** the range uses UTF-16 units from the owning source without counting line terminators

### Requirement: Dependency buffer lifecycle
Configured open buffers SHALL override saved source. Changes SHALL refresh open
document diagnostics. Closing a configured document SHALL reread only its allowed
path and restore saved contents; a reread failure MUST NOT silently use stale
text. File URI matching SHALL preserve platform path case semantics and MUST NOT
read arbitrary URIs. Opening two aliases for one configured file simultaneously
SHALL be rejected. External changes to closed files MAY require a server restart.

#### Scenario: Correct dependency and clear stale errors
- **WHEN** an invalid configured buffer is replaced by valid source
- **THEN** dependent diagnostics refresh and obsolete source diagnostics clear

#### Scenario: Save or discard and close
- **WHEN** a configured buffer closes
- **THEN** dependent checks use its current saved contents and omit its former open version

### Requirement: Independent versioned checks
Each refresh SHALL discard temporary state. Open roots MUST NOT enact themselves
into other roots' context. Counsel SHALL aggregate shared dependency errors and
clear obsolete diagnostics. It SHALL publish accepted document versions and
ignore stale versioned edits. Cancellation SHALL reach temporary Court work.

#### Scenario: Two roots share one bad dependency
- **WHEN** one importer closes and another remains open
- **THEN** the remaining importer's dependency diagnostic is retained

#### Scenario: Stale change arrives
- **WHEN** a change has a version no newer than the accepted version
- **THEN** accepted text and its diagnostics remain unchanged

#### Scenario: Checked root statute is not configured
- **WHEN** a valid open S-1 root is checked before another root imports its name
- **THEN** the check does not make that statute available to the other root
