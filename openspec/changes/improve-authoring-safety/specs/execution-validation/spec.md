## ADDED Requirements

### Requirement: Reject malformed execution data safely
The Court SHALL reject missing required recovery payloads and negative execution cursors before dereferencing or indexing. It SHALL reject negative collection/call arities and preserve the existing operand-underflow verdict before allocation. Valid zero arities and legitimate legacy replay normalization MUST remain supported.

#### Scenario: Damaged recovery history
- **WHEN** a PUSH lacks its value or an attention cursor is negative
- **THEN** recovery returns a contextual error instead of panicking

#### Scenario: Invalid instruction count
- **WHEN** a stored collection or indirect-call instruction requests negative or unavailable operands
- **THEN** execution produces the appropriate verdict without an invalid or oversized allocation

#### Scenario: Valid existing history
- **WHEN** ordinary or legacy selective-receive history is recovered
- **THEN** replay and zero-arity behavior remain valid
