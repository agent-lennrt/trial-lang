# Design

Parse documents normally. Without imports use the existing compiler. With
`--canon`, build a fresh memory-backed statute context from embedded sources;
compile the document only when every declared import exists in that context.
Use Court's existing File/Enact validation rather than duplicating dependency
traversal. Release the local context and any temporary filing after each check;
never execute the document.

Keep syntax-only fallback when custom dependencies are unresolved. Do not
discover files or connect to a broker. Existing synchronous parsing and document
limits remain in effect.

Preserve parse ranges. For imported compilation failures, compare the rejection
against independent compilation of the original source. Exact matching
particulars/line/column establish a local diagnostic location. Otherwise emit a
zero-width document-start range with explicit canon-import context. Never use
the independent rejection when authoritative import-aware checking succeeded.

This deliberately postpones cross-file source maps, workspace configuration,
and compiler caching. Fresh contexts avoid accumulated cases and stale edits.
