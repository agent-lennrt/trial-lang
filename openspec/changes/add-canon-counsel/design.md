# Design

Parse documents normally. Without imports use the existing compiler. Build one
fresh memory-backed statute context per refresh from enabled canon and explicit
sources. Use Court's read-only Check to validate roots without duplicating
dependency traversal or publishing checked roots. Release the context after the
refresh; never execute documents.

Without explicit sources, keep syntax-only fallback when custom dependencies are unresolved. Do not
discover files or connect to a broker. Existing synchronous parsing and document
limits remain in effect.

Preserve parse ranges. The accompanying `add-custom-statute-counsel` proposal
replaces the initial independent-compilation heuristic with compiler source
ownership. Imported failures use their owning text and URI, including virtual
bundled-source locations. Its read-only Court check also avoids creating cases
or enacting checked roots. Fresh contexts avoid accumulated cases and stale edits.
