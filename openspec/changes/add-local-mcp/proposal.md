# Use the Advocate without Kafka

## Why

The MCP CLI opens Kafka before initialization, so clients without a broker
cannot use even its isolated testing capability. Existing memory-backed tool
tests demonstrate that the same tool suite can support a local workflow.

## What Changes

- Add explicit `trial mcp --local` with one temporary log per process.
- Preserve all existing tools, schemas, and results.
- Identify ephemeral state in initialization instructions.
- Reject conflicting explicit broker options and ignore ambient broker settings.
- Verify real-process workflows, session isolation, EOF, and signal cancellation.

## Capabilities

### New Capabilities

- `local-mcp`: Run the existing MCP tools with temporary session storage.

### Modified Capabilities

None. The default Kafka mode and stored formats remain unchanged.

## Impact

This adds a CLI option and mode-specific initialization instructions. Update the
interface and threat-model documentation; language and bytecode specifications
in `spec/` are unchanged. See [ADR 0010](../../../docs/adr/0010-local-mcp-sessions.md).
