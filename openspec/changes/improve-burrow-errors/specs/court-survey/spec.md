## ADDED Requirements

### Requirement: Complete court-wide survey reads
The court-wide survey SHALL return an error and no report when a required
commencement-ledger, archive, catalog, attention, or motion-record read fails.
The error SHALL preserve its underlying cause and identify the failed stage.

#### Scenario: A catalog cannot be read
- **WHEN** an archive contains documents and its catalog read fails
- **THEN** the survey returns the read error instead of reporting the documents as drafts

#### Scenario: A residual observation cannot be read
- **WHEN** a commencement-ledger, archive, attention, or motion-record read fails
- **THEN** the survey returns the read error instead of omitting that observation

#### Scenario: A read is canceled
- **WHEN** cancellation interrupts a required survey read
- **THEN** the returned error retains the cancellation cause and no partial report is returned

#### Scenario: All required reads succeed
- **WHEN** every case audit and residual-state read succeeds
- **THEN** the survey reports the existing audit results, drafts, unconvened cases, and spent motions
