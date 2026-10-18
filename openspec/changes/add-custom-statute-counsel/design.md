# Design

The CLI reads only explicitly configured files. Normalize each path to an
absolute file URI and reject duplicate normalized paths. Standard input belongs
to the protocol. Enforce 100 sources, 4 MiB per source, and 256 MiB aggregate.
Keep an exact URI-to-path allowlist for bounded reloads on document close.

Counsel retains configured source order, with canon first. An open document with
a matching URI overrides its configured source. Closing it reloads the saved
file. A reread failure becomes a diagnostic instead of reviving stale source.
If distinct inputs name the same statute, the last successful enactment supplies
imports; invalid inputs remain diagnosed even when an earlier version exists.
No watchers, import-driven discovery, or arbitrary URI reads are introduced.

Each refresh builds fresh temporary dependency state. Read-only Court.Check
validates open K-1/S-1 roots against that state without enacting roots, allocating
cases, or executing instructions. Explicit sources make this a complete context;
without them the existing unknown-import syntax-only fallback remains.

Parser and compiler ownership belongs to whole top-level declarations, articles,
and offices. Nested errors inherit the owner. Deferred patches and static
examination retain that owner. Court stamps imports with the statute identity;
Counsel maps it to configured file text and URI, or a trial-canon virtual URI.
Error strings, bytecode positions, and serialized data remain unchanged.

Refresh diagnostics across open roots, deduplicate shared failures, and clear
obsolete URI buckets. Keep accepted versions separately from text and ignore
stale versioned changes. Calculate UTF-16 positions from the owning source,
including CRLF and supplementary Unicode. Related information links importers.

Alternatives rejected: line-number rebasing changes instruction positions;
duplicated import resolution drifts from Court; startup snapshots on close lose
normal save/close behavior; publishing open roots contaminates other checks;
watchers and caches add unrelated invalidation and filesystem scope.
