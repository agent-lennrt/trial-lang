# ADR 0012: Custom statute context and diagnostic source ownership

Date: 2026-10-18

Status: proposed

## Context

Counsel can validate bundled canon imports, but has no explicit context for
custom statutes. Imported compiler errors carry line numbers without source
ownership. ADR 0011 therefore reports ambiguous errors at document start.
Authors need to edit dependencies and locate the actual source of a rejection.

## Decision

Add repeated `trial counsel --enact <path>` options alongside `--canon`.
Only explicitly configured statute files enter the dependency context. Open
editor text for a configured source takes precedence over its saved contents.
Recheck affected open documents when dependency text changes and clear stale
diagnostics. Do not discover files from import names or arbitrary document URIs,
contact Kafka, or execute editor documents. Keep source/count/aggregate bounds.

Preserve optional source ownership in Gregor's parsed top-level declarations,
articles, and offices. Every nested statement belongs to its enclosing owner.
Parser, static examiner, immediate compiler errors, and delayed referral/call
errors carry that ownership in rejection metadata. Court incorporation stamps
imported declarations with their statute identity. Counsel maps that identity
to its configured file URI and converts byte columns against the correct text.

Diagnostic ownership is metadata only. Exclude it from JSON, keep rejection
error strings unchanged, and leave instruction positions, serialized bytecode,
Kafka records, and the public canon API unchanged. Reuse Court's incorporation
rules rather than creating a separate editor compiler or dependency resolver.

Build one fresh configured statute context per refresh. Add an internal,
read-only Court check that parses, validates the K-1/S-1 form, incorporates, and
compiles without creating topics or enacting the checked root. This lets open
statutes share configured dependencies without publishing one open root into
another root's context. No public Go API is added.

Report dependency errors at the actual file/range, with importing locations
where helpful. Track diagnostics across open documents so edits and closes do
not leave stale or accidentally cleared dependency errors. Default Counsel
without configured dependency context retains its existing behavior.

## Consequences

Authors can configure project statutes explicitly and receive correct
cross-file errors, including transitive dependencies. Source provenance replaces
the heuristic document-level fallback for known sources from ADR 0011.

Configured file access and editor overlays introduce lifecycle and resource
boundaries. These must be documented and tested, including missing files,
invalid sources, aliases, source limits, dependency edits, and diagnostic clearing.
No background workspace scanning or arbitrary file access is authorized by a
protocol document URI.
