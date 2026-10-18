# Check canon programs while editing

## Why

Counsel currently suppresses compilation whenever a document imports a statute.
This hides known compilation errors even when all dependencies are bundled.

## What Changes

- Add opt-in `trial counsel --canon` using only embedded statute sources.
- Compile fully resolvable imports through the existing Court and Gregor paths.
- Preserve default behavior and syntax-only fallback for unknown custom imports.
- Report imported-source failures with honest document-level diagnostic ranges.
- Avoid executing documents and retain no temporary cases across edits.

## Capabilities

### New Capabilities

- `canon-counsel`: Compile known bundled imports during editor diagnostics.

### Modified Capabilities

None. Default LSP behavior and language/storage formats remain unchanged.

## Impact

The CLI gains a language-server option. Documentation describes its dependency
scope and conservative source locations. Detailed language rules in `spec/`
remain unchanged. See [ADR 0011](../../../docs/adr/0011-canon-editor-diagnostics.md).
