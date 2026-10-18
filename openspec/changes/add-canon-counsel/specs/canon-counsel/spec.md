## Purpose

Check compilation of editor documents whose imports can be resolved entirely
from the bundled canon, while preserving honest diagnostic locations.

## ADDED Requirements

### Requirement: Explicit bundled import context
`trial counsel --canon` SHALL compile a parsed document when every declared
import names a bundled statute. It SHALL use existing incorporation and compiler
rules without executing the document. Default mode SHALL remain unchanged.
When no explicit statutes are configured, unknown or mixed custom imports SHALL
retain syntax-only diagnostics.

#### Scenario: Invalid referral with a known import
- **WHEN** a canon-enabled document imports a bundled statute and refers to a missing article
- **THEN** Counsel reports the compilation error at its owning local source range

#### Scenario: Valid transitive imports
- **WHEN** a document uses offices provided by a bundled statute and its dependencies
- **THEN** Counsel resolves them without false undefined-office errors

#### Scenario: Unknown custom dependency
- **WHEN** any import is outside the bundled canon and no explicit statutes are configured
- **THEN** Counsel checks syntax without claiming complete compilation or fetching external sources

### Requirement: Report diagnostic locations honestly
Parse diagnostics SHALL retain UTF-16 source ranges. Imported compilation
failures SHALL use the owning source's text and URI, as specified by the
accompanying custom-statute-counsel capability. Bundled sources SHALL use stable
virtual URIs rather than treating imported lines as local source lines.

#### Scenario: Local name collides with canon
- **WHEN** compilation reports a collision originating from an appended canon declaration
- **THEN** Counsel reports the canon declaration's virtual URI and range instead of treating its line as local

### Requirement: Fresh document checks
Checking SHALL NOT execute source or retain cases between checks. Each imported
document diagnostic pass SHALL use temporary dependency and filing state that is
discarded after the check. Existing document and framing limits SHALL remain in effect.

#### Scenario: Document changes after an error
- **WHEN** an editor replaces a rejected canon program with a valid one
- **THEN** the new diagnostics clear the error without retaining the old program or its temporary case
