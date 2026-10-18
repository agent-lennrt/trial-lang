# ADR 0009: Compile-only local validation

Date: 2026-10-18

Status: proposed

## Context

Authors can run source locally, test it with a deposition, or file it with
Kafka. They need a direct way to check a program and its imports without
executing statements or writing a deposition.

## Decision

Add `trial check <source.trial>` for Form K-1 cases and Form S-1 statutes.
Use the existing filing and enactment paths on a new in-memory log. Never
proceed with a case or start child workers. Reject Form K-2: an amendment
needs the existing case to validate its article and office context.

Accept `--canon`, repeated `--enact` paths in dependency order, and a positive
`--timeout` with a 30-second default. Load the bundled canon before explicit
statutes. Use the same source limits and process cancellation grace as
[local execution](0004-brokerless-run.md). Resolve paths from the current
directory and allow `-` for only one source, including dependencies.

Return status 0 and `Source accepted.` on stdout after validation. Return
status 1 for source, compilation, output, timeout, or cancellation errors, with
diagnostics on stderr. Return status 2 for invalid arguments. No case number or
persistent state survives the command.

## Consequences

Editors and CI can validate source without a broker or runtime effects. The
result covers parsing, compilation, and incorporation using the supplied
statutes. It does not establish successful execution, input availability, or
termination. Runtime errors can remain in accepted source. The command is not
a sandbox or a total-memory limit.

Using filing and enactment avoids a separate implementation of compiler and
dependency rules. Temporary in-memory records are discarded on exit. Existing
commands, stored formats, and language rules are unchanged.
