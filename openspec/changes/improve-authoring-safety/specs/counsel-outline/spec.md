## ADDED Requirements

### Requirement: Outline current open source
Counsel SHALL advertise documentSymbolProvider and return source-ordered symbols for named declarations, articles, offices and nested sections using the current open buffer. Ranges SHALL use UTF-16 positions and contain child ranges. The method MUST NOT execute code or load arbitrary files.

#### Scenario: Open and changed source
- **WHEN** a valid buffer is opened or changed
- **THEN** the outline reflects that accepted buffer and preserves office/section hierarchy

#### Scenario: Missing or syntactically invalid source
- **WHEN** the requested buffer is closed, absent, empty or fails parsing
- **THEN** the method returns an empty symbol list

#### Scenario: Unicode and CRLF
- **WHEN** source contains supplementary Unicode or CRLF endings
- **THEN** reported ranges address that source in UTF-16 units
