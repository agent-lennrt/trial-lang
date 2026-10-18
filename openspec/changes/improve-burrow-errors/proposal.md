# Report incomplete Burrow surveys

## Why

The court-wide audit silently ignores failures while reading residual state.
A catalog read failure can classify cataloged documents as drafts. Other read
failures can omit commenced cases or spent motions from a successful report.

## What Changes

- Return a contextual error when any residual-state read fails.
- Preserve the underlying error and return no partial survey as a success.
- Cover each read stage with deterministic failure-injection tests.

## Capabilities

### New Capabilities

- `court-survey`: Require complete reads before reporting a court-wide survey.

### Modified Capabilities

None.

## Impact

The CLI's existing error path reports an incomplete survey. Successful report
contents, language behavior, storage formats, and exported Go APIs do not change.
