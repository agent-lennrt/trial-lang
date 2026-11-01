# ADR 0014: Bounded authoring and stored execution validation

Date: 2026-11-01

Status: proposed

## Context

Independent probes found corrupt stored execution data can panic recovery or
request invalid allocations, Kafka producers impose an undocumented lower
record bound, and JSON escaping can make an otherwise useful MCP output page
exceed its envelope limit. Authors also lack source outlines, bounded syntax
depth, selectable deposition suites and machine-readable test results.

## Decision

Validate required stored event payloads, recovered cursor ranges and instruction
arities before dereferencing, indexing or allocating. Return the established
storage error or execution verdict. Preserve legitimate legacy replay state,
stored JSON tags, instruction positions and ordinary program behavior.

Configure every Kafka producer for the existing accepted record bound including
encoding overhead. Explicitly document the separate broker/topic limit. Budget
MCP observe pages against their encoded result envelope and return a resumable
prefix, preserving an explicit error for one record that cannot fit.

Bound recursive syntax and AST depth with a documented fixed limit and positioned
source-aware rejection. Cover left-associated expression trees as well as syntax
nesting. This is a resource bound, not a change in evaluation order or bytecode.

Extend Counsel with document symbols derived from the existing parsed AST and
open buffer. Provide source-ordered UTF-16 ranges without execution or additional
filesystem discovery. Diagnostics remain the channel for invalid source.

Add opt-in JSON Lines deposition results with per-file outcomes and a summary.
Keep human output the default. Independently support regex selection, listing,
lexical path deduplication and opt-in empty-suite failure. Listing never executes
programs. Cancellation and output failures must remain explicit.

## Consequences

Each boundary needs fail-first behavioral checks, exact/over-limit cases where
applicable, user-facing documentation and normative specification updates.
No public Go API or persisted record shape is added. Release preparation retains
ADR0013 owner approval and gains Go-module-major compatibility and complete
subprocess cancellation. Toolchain patch versions may advance to fix a reachable
standard-library vulnerability; module and Action updates stay separate.
