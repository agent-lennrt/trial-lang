# ADR 0010: Run explicit local MCP sessions

Date: 2026-10-18

Status: proposed

## Context

The Advocate can test programs on an isolated memory log, but `trial mcp` first
connects to Kafka. A client without a broker cannot initialize the server or use
its local test tool. The existing tools already support the memory log used by
their tests and the local CLI workflows.

## Decision

Add `trial mcp --local`. Create one fresh in-memory log for the process and pass
it to the existing Advocate server. Expose the same tools and result formats.
Do not open a broker connection. Ignore `TRIAL_BROKER` in this mode; reject an
explicit `--broker` option together with active `--local` as invalid arguments.

Initialization instructions identify the temporary storage. Cases, statutes,
and records last only for that process. Case identifiers cannot resume work in
a later process. `trial_test` continues to create a separate isolated log and
does not inherit statutes from the local session.

The CLI owns and closes its session log. EOF ends the server normally. Existing
signal cancellation interrupts input and active Court work. Tools retain their
existing size and execution bounds. `trial_proceed` processes the named case;
this mode does not add automatic docket workers.

## Consequences

Clients can rehearse the complete MCP workflow without Kafka. The default mode
still opens and verifies Kafka before serving requests. Broker-backed recovery,
retention, and persistence behavior do not change.

The memory session is temporary storage, not a security sandbox or a total heap
limit. Long sessions can retain substantial data. Memory-backed success does
not establish Kafka durability or recovery behavior.

A test-only tool subset would require separate discovery and dispatch rules.
Lazy broker connections would add connection lifecycle and retry decisions.
The explicit local mode avoids those changes and keeps persistence a visible
choice made by the process owner.
