## Context

The local run command already owns bounded source loading and a cancellation
grace. Court filing and enactment already implement dependency validation.

## Goals / Non-Goals

Support compile-only checks of one K-1 or S-1 source with explicit dependencies.
Do not execute instructions, inspect Kafka, validate K-2 without a case, or
claim that accepted programs run successfully.

## Decisions

Use a fresh MemoryLog and existing File/Enact operations. Share source loading
with run so canon order, size bounds, and read cancellation stay consistent.
Keep command-specific diagnostics while sharing the process cancellation grace.
Use the same 30-second default timeout, 4 MiB source bound, 100 explicit
statutes, and 256 MiB aggregate explicit-statute bound as run.

Reject a standalone K-2 explicitly rather than pretending it has an empty case
context. Emit a fixed success line without source-derived text or identifiers.

## Risks / Trade-offs

Compilation can outlive context cancellation internally; the process action
retains the existing one-second grace before exit. Library-style test helpers
remain synchronous and do not retain caller-owned writers. Temporary filing
records cost memory but reuse the authoritative validation path.
