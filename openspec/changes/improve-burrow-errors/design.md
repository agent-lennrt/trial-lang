# Design

## Context

`SurveyBurrow` combines individual case audits with commencement, archive,
attention, and motion observations. Its error result already represents an
incomplete operation, but several observation reads currently ignore errors.

## Goals / Non-Goals

A successful result must include every requested read. Storage and cancellation
errors must remain recognizable by callers. Existing successful classifications
and record parsing behavior remain unchanged.

This change does not add partial reports, concurrent snapshots, or new record
validation. A survey still observes topics in sequence while the court may run.

## Decisions

### Stop when an observation cannot be read

Return `nil` and a wrapped error containing the case and observation stage.
Do not guess whether an unavailable topic is empty. Case topics are created
together, so a missing topic is also an incomplete survey.

### Preserve the current report and error boundaries

No new fields, stored bytes, or public package APIs are needed. The existing
error return and CLI failure path carry the failure. Successful reports retain
their existing order and residual-state definitions.

## Risks / Trade-offs

A caller that previously received incomplete results now receives an error.
This is intentional: the report has no field to identify skipped observations.
Tests inject failures specifically after the case audit where necessary, so
passing tests cannot merely exercise the audit's existing error handling.
