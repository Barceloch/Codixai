# Pull request

## What this changes

One paragraph. The objective of the change, in the terms of the task it
belongs to.

**Task:** the identifier of the task in `docs/tasks/`, for example `P1-02`.
If the change has no task, say so and explain why.

## Scope

- What the change includes.
- What it deliberately does not include, and the task that covers it.

## Type of change

- [ ] Bug fix
- [ ] New rule
- [ ] New language support
- [ ] Architecture or contract change (an ADR is required)
- [ ] Documentation
- [ ] Tests only
- [ ] Build or CI

## Checks executed

The commands you ran, with their real result. A command that was not run is
not listed.

```bash
gofmt -l .   # result
go build ./...
go vet ./...
go test ./...
```

## The self check of the agent

The result of `codixai audit --diff --worktree` over this change, and the
findings that you consider false positives, with their evidence.

## ADR

The ADR that this change requires, or why it does not require one.

## Checklist

- [ ] The change stays inside the scope of the task.
- [ ] The tests and the documentation of the change travel with it.
- [ ] The standard output still holds only the requested document, and
      `schema_version` is unchanged, or the change is intentional and
      documented.
- [ ] The support matrix of every affected rule matches the fixture corpus.
- [ ] No telemetry, no upload of code and no remote execution were added.
- [ ] The commit messages follow Conventional Commits and carry no AI
      attribution.
- [ ] No secret, token or credential appears in the diff, in the tests or in
      the documentation.
