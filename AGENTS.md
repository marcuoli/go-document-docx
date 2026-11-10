# Repository guidance

This is a reusable document library. New APIs live in cohesive functional directories, with focused tests and runnable examples under examples. Keep product policy and application identities outside this repository.

Use Go 1.27.3 and official released dependencies. Reuse shared atomic operations. Do not modify or copy third-party dependency implementations. Comments and examples must exclude originating or consuming application names, private endpoints, customer material and secrets.

Every exported symbol needs useful Godoc starting with its name. Explain units, ownership, mutation, errors, concurrency, limitations and partial success where relevant. Explain non-obvious choices beside the implementation and update nearby comments when behavior changes. See CONTRIBUTING.md.

Track bounded changes in the source forge with typed imperative titles, one type label and the exact heading ## Validation and acceptance. Use codex/<type>/<issue>-<description> branches, changelog-bearing reviewed PRs and an independent review before integration. Gitea owns history; GitHub is the public distribution copy. Audit all public source and commit metadata before public pushes. Module tags are immutable semantic versions.

Canonical source stays on I:. Generated evidence, caches, scratch and temporary issue-owned worktrees use the owning project-first V: namespace. Record the actual Git admin path, preserve unique evidence and remove finished temporary worktrees through Git before issue closure. Preserve unrelated files and credentials.
