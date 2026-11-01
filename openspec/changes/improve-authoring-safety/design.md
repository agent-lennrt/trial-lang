## Context

The compiler, Court, Log adapters, Advocate and Counsel already define the
authoritative execution and parsing paths. Extend those paths and keep each
decision in one place. Do not create another parser or test runtime.

## Decisions

Use structural validation at stored-data boundaries and check arities against
available operands before allocating. Keep ordinary semantic underflow verdicts
and legitimate replay normalization. Kafka producer constructors share a bounded
configuration that accounts for both key and value plus encoding overhead.

MCP page sizing uses the actual serialized tool-result envelope. A smaller
successful prefix is preferable to losing all available records, while a single
unrepresentable record remains an explicit error. Preserve cursor semantics.

Document outlines use currently open source and existing AST metadata. Derive
UTF-16 ranges from that buffer, maintain hierarchy/source order, and perform no
execution or arbitrary file loading. Resource validation covers both recursive
syntax and expression-tree depth before compiler traversal.

Separate suite selection from result presentation. Selection operates in text
and JSON modes, listing performs discovery only, and first-occurrence lexical
path deduplication avoids accidental duplicate execution. JSON reporting streams
results and a summary; cancellation and output failures are explicit.

## Verification and limits

Each behavioral fix has a negative control against prechange code where
practical. New capabilities have real CLI/protocol checks. Preserve bytecode,
stored JSON and canon API regressions. Full checks include race, fuzz, examples,
workflow/spec/API validation and vulnerability scanning. Live Kafka evidence
requires the hosted broker job if Docker is unavailable locally.

One independent integrated review follows the initial implementation phase and
one consolidated repair cycle. Quality-tool findings require concrete behavioral
interpretation; agreement and passing limited tests do not prove universal
correctness. Companion metrics and mutation artifacts stay outside the diff.
