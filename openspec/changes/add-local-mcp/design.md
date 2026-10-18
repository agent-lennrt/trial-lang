# Design

## Mode selection

Parse `--local` before opening storage. In local mode reject any explicitly
supplied `--broker` option; environment broker defaults have no effect. Create
one MemoryLog and retain it until the CLI exits. Default mode continues using
the existing eager Kafka open and health check.

## Protocol and ownership

Pass the chosen log to the existing Advocate. A minimal mode flag selects
initialization instructions that accurately describe storage lifetime. Keep
discovery, tool validation, dispatch, and JSON result formats unchanged.

The CLI owns the log and protocol input lifecycle. EOF returns success; existing
context cancellation closes the input adapter and reaches running Court calls.
The caller still owns stdout. No background docket worker or new timeout model
is added. Existing tool bounds apply; no total heap bound is claimed.

## Isolation and alternatives

Each new process receives an empty log. The `trial_test` tool remains separately
isolated from the session log. A test-only mode would need a second tool surface;
lazy Kafka connection would add connection-state and retry behavior. Reusing
the full memory-backed tool suite keeps the boundary smaller.
