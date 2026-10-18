## Why

Authors cannot compile source and resolve its imports from the CLI without
executing it, creating a deposition, or connecting to Kafka. A local check
supports editors and CI before the program is ready to run.

## What Changes

- Add `trial check` for K-1 cases and S-1 statutes with local dependencies.
- Reuse bounded source loading and cancellation from `trial run`.
- Document the distinction between filing acceptance and runtime success.

## Capabilities

### New Capabilities

- `local-validation`: compile-only CLI validation with isolated state.

### Modified Capabilities

None. Language and storage rules remain in `spec/` unchanged.

## Impact

CLI registration, help, source preparation, tests, and interface documentation.
No new dependency, public Go API, Kafka access, or stored format. See ADR 0009.
