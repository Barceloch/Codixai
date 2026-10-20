---
name: Bug report
about: Report a defect of Codixai with a reproducible case
title: ""
labels: bug
assignees: ""
---

## What happens

A short description of the defect.

## How to reproduce

The exact commands, starting from a clean state.

```bash
# the commands you ran
```

The repository layout that the reproduction needs. A reduced example is
enough; do not attach a private repository.

## Expected result

What you expected the tool to do.

## Actual result

What the tool did. Include the output and the exit code.

```text
# the output of the tool
```

## Environment

- Codixai version: `codixai version`
- Go version (if you built from source): `go version`
- Git version: `git --version`
- Operating system and version:

## Notes

Anything else that helps: the configuration file of the repository
(`.codixai.yml`), the language of the affected files, a related issue.

**Never include secrets, tokens, credentials or the content of a private
repository.** A reduced reproduction is enough to fix a defect.
