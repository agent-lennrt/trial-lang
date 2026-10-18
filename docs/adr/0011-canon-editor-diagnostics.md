# ADR 0011: Check bundled canon imports in Counsel

Date: 2026-10-18

Status: proposed

The source-location heuristic below is superseded by
[ADR 0012](0012-custom-statute-editor-diagnostics.md), which also adds explicit
custom dependencies. The original opt-in canon and unknown-import fallback
remain available.

## Context

Counsel checks only syntax when a document imports statutes because it has no
dependency context. This also hides ordinary compilation errors in documents
using the bundled canon. The CLI now offers explicit canon-aware source checks.

## Decision

Add opt-in `trial counsel --canon`. Use the embedded canon as the only import
context. If every declared import names a bundled statute, resolve dependencies
through the existing Court incorporation path and compile with Gregor. Otherwise
retain syntax-only diagnostics. The default command remains unchanged.

Create temporary storage for each imported-document check and use the existing
Court filing/enactment checks. Do not execute documents, retain cases between
edits, read workspace files, or contact Kafka. Temporary filings and their
identifiers are discarded after the check. Existing document and protocol
limits still apply.

Imported AST nodes retain their original line numbers. Treat a compile failure
as a local source location only when compiling the original source independently
returns identical rejection particulars, line, and column. All other import
validation errors use a zero-width range at document start and explicitly say
they arose while checking bundled canon imports. Parse errors keep their normal
UTF-16 ranges. A standalone rejection never overrides successful import checking.

## Consequences

Editors can catch missing articles, wrong calls, and name collisions in canon
programs without a broker. The command validates source; it does not execute it
or establish runtime safety. Unknown custom imports still lack compile checks.

Some imported compilation errors have document-level locations. This is more
honest than highlighting a bundled file's line in the user's document. Full
cross-file source mapping and custom statute configuration are separate work.

Temporary storage avoids stale edits and accumulated cases. It adds the cost of
loading five small bundled statutes to imported-document checks. No persistent
storage, new dependencies, or public Go API is introduced.
