# Repository Guidance

Keep this repository idiomatic, maintainable Go. Prefer small, focused packages
and files with clear responsibilities over broad accumulation points.

## Architecture

Use a lightweight ports-and-adapters shape:

- Core/domain code owns reusable behavior and should not depend on UI, command,
  form, handler, or transport details.
- Application code coordinates state, validation, and use cases through small
  APIs.
- Adapter code translates external framework events, commands, forms, storage,
  or item metadata into application/core calls.

When adding a feature, put reusable behavior behind the appropriate core or
application boundary first. Keep framework-facing adapters thin.

## File Organisation

Group code by feature or responsibility. Avoid broad files that collect many
unrelated concerns.

Do not create or expand files that contain, for example:

- every command in a package,
- every form in a package,
- every handler plus parsing plus business logic,
- unrelated helper functions gathered into a generic dumping ground.

If a file starts mixing responsibilities or becomes hard to scan, split it by
feature, use case, or adapter boundary before adding more code.

## Configuration and Dependencies

Prefer explicit Go configuration/options for library behavior. Add config files
or third-party config libraries only when there is a real runtime configuration
need for downstream users.

Do not add dependencies for small conveniences. Reuse the standard library and
existing project helpers first. Any new dependency should have a clear runtime or
maintenance benefit.

## Testing and Verification

For behavior-preserving refactors, lock behavior with existing or new regression
tests before moving logic.

Before claiming completion, run the relevant local gates. For normal code
changes, use:

```sh
go test ./...
go vet ./...
golangci-lint run ./...
```

For concurrency-sensitive or broad changes, also run:

```sh
go test -race ./...
```

Prefer targeted core/application tests over heavy adapter integration tests when
the same behavior can be verified without framework setup.

## Local agent workflow notes

- PRs, pushes, and commits for this checkout target `cjmustard/we` by default.
  Verify the target before opening a PR, for example:

  ```sh
  gh repo view cjmustard/we --json nameWithOwner,defaultBranchRef
  ```

- `AGENTS.md` and `plan.md` are local-only guidance files in this checkout.
  Never stage, commit, push, or include them in pull requests unless the user
  explicitly overrides this rule in the same request.

## API boundaries and wrappers

Avoid type aliases, forwarding wrappers, and compatibility shims that only hide a
package move or add another name for the same behavior. Do not add patterns like:

```go
type SomeConfig = otherpkg.SomeConfig

func DoThing(...) error {
	return otherpkg.DoThing(...)
}
```

unless they provide a real compatibility boundary, reduce caller churn during a
migration, preserve a public API intentionally, or make an adapter boundary
clearer. Prefer moving callers to the owning package when the wrapper is not
pulling its weight.

## Shared metadata

Keep supported values, option lists, and behavior metadata in one owning package.
Avoid copying parallel string lists across adapters and services (for example one
list for forms and another switch/list for guardrails). Prefer a single
metadata table plus small accessors or predicates, and let adapters read from
that owner.
