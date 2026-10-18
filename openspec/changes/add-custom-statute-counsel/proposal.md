# Edit custom statutes with accurate source diagnostics

## Why

Counsel has no explicit custom statute context. Imported declarations keep line
numbers but lose file ownership, forcing ambiguous errors to document start.
Authors need to edit a dependency and see errors in the file that owns them.

## What Changes

- Add ordered, bounded `trial counsel --enact <path>` inputs alongside `--canon`.
- Use configured open buffers as overlays and reload allowed files on close.
- Preserve diagnostic source ownership through parsing, compilation, and imports.
- Publish accurate cross-file ranges, related locations, versions, and clearing.
- Reuse a read-only Court check without executing or publishing open roots.

## Capabilities

### New Capabilities

- `custom-statute-counsel`: Explicit editor dependencies and source diagnostics.

### Modified Capabilities

None in the accepted specification. This proposal extends the pending
`add-canon-counsel` proposal and replaces its conservative location heuristic.

## Impact

CLI configuration, LSP diagnostic fields, and internal source metadata change.
Language semantics, bytecode, stored JSON, and the public canon API remain
unchanged. See [ADR 0012](../../../docs/adr/0012-custom-statute-editor-diagnostics.md)
and the normative [incorporation rules](../../../spec/spec.md#132a-incorporation-by-reference).
