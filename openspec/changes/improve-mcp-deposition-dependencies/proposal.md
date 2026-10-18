# Test imported offices through MCP

## Why

`trial_test` cannot execute a deposition containing `ENACT`: the tool accepts no
statute sources, and the isolated deposition runner cannot use broker state.
This prevents clients from testing imported offices before filing a case.

## What Changes

- Add optional inline `enactments` keyed by exact deposition `ENACT` labels.
- Resolve sources in declaration order, retaining repeated declarations.
- Reject missing, unused, malformed, and excessive inputs before execution.
- Keep tests isolated from the filesystem and the Advocate's log.

## Capabilities

### New Capabilities

- `mcp-deposition-dependencies`: Supply bounded inline statutes to isolated tests.

### Modified Capabilities

None. Existing tool results, deposition syntax, and stored formats stay the same.

## Impact

The optional MCP input is an additive wire-interface change. Existing tests
without dependencies keep their behavior. The updated interface reference and
deposition guide define label resolution; language and bytecode rules in `spec/`
are unchanged. See [ADR 0008](../../../docs/adr/0008-mcp-deposition-dependencies.md).
