# ADR 0008: Supply deposition statutes inline through MCP

Date: 2026-10-18

Status: proposed

## Context

The Advocate asks clients to test programs before filing them. Its `trial_test`
tool accepts program and deposition source, but cannot load the statutes named
by `ENACT`. Any test that imports an office therefore lacks a complete local
workflow. Using the server's enacted statutes would make a test depend on broker
state and violate its isolated-memory contract.

## Decision

Add an optional `enactments` object to `trial_test`. Its keys are the exact labels
in the deposition's `ENACT` statements; its values are inline statute sources.
These labels are lookup keys, even when they resemble filesystem paths. Resolve
them in declaration order. A repeated label uses the same source at each position.
Do not access files, the broker, or the server's log.

Reject missing or unused keys, null or non-string values, more than 100 entries,
sources larger than 4 MiB, and an expanded source sequence larger than 256 MiB.
Count repeated declarations toward that sequence bound. Validate the complete
mapping before execution. These failures use existing failed-tool results.
An invalid statute remains a deposition contradiction, as in `trial test`.

The existing 16 MiB JSON request limit also applies, including escaped strings
and the two required sources. Omitting `enactments` is valid when the deposition
has no `ENACT` statements. Empty source strings are explicit inputs and reach
the ordinary statute parser. Result fields and execution limits stay unchanged.

## Consequences

MCP clients can test custom and transitive imports without writing files or
creating persistent statutes. Clients supply each distinct label once and put
dependency order in the deposition itself. Unused sources fail so a misspelled
label cannot silently disappear from the test.

No new dependency, language syntax, stored format, or filesystem trust boundary
is introduced. Canon sources use the same explicit mechanism; there is no
implicit broker lookup or bundled-canon option.
