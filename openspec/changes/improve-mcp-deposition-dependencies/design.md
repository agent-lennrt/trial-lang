# Design

## Source ownership and resolution

The MCP caller owns every source. The `enactments` object supplies one source for
each distinct `ENACT` label. Labels match exactly; they never cause filesystem
reads. The deposition owns execution order, including repeated declarations.
Resolve all labels before assigning the runner's `EnactSources`.

## Validation and execution

Validate object shape, required strings, 100-entry count, 4 MiB source bounds,
and the runner's 256 MiB expanded-source bound before executing. Repeated labels
count repeatedly toward the latter bound. The existing transport limit is
16 MiB for the entire JSON request and can be reached first.

Argument failures use `isError`. Statute parsing or enactment failures use the
existing `consistent: false` deposition result. The runner creates its own
memory log and closes its child workers; the server log is never consulted.
Existing caller cancellation and deposition execution allowances remain intact.

## Alternatives

Reading labels as filesystem paths would expand the MCP trust boundary and make
tests depend on the server's working directory. Looking up statutes in the
server log would violate test isolation. A positional source array would hide
missing or accidentally reordered dependencies. A map with exact labels makes
those errors explicit while retaining declaration order in one place.
