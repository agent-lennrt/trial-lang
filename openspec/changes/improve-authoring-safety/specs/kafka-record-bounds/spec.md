## ADDED Requirements

### Requirement: Match accepted record bounds in every producer
All Kafka producer constructors SHALL configure an encoded batch limit sufficient for one accepted maximal key and value plus encoding overhead. Broker and topic limits remain separately required; clients MUST NOT claim to override them.

#### Scenario: Large configured topic
- **WHEN** a topic permits a valid record larger than one MiB
- **THEN** Append, AppendBatch and Commit can roundtrip that record through Kafka

#### Scenario: Constructor consistency
- **WHEN** an ordinary or transactional producer is created
- **THEN** its actual configuration accommodates the existing key/value bounds
