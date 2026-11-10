# Contributing

Use Go 1.27.3 and functional package imports. Keep changes focused and use only official compatible releases.

## Comment convention

Start exported-symbol comments with the symbol name. Explain the observable contract, zero values, inputs and units, ownership, mutation, concurrency, error identity, side effects and partial success when relevant. State limitations precisely; do not promise rendering, font availability or durability beyond the implementation.

Place comments beside non-obvious decisions and explain the concrete problem and chosen behavior. Keep comments current. Avoid obvious restatements, private paths, customer values, secrets, transient investigation notes and originating or consuming application names. Examples use synthetic inputs.

## Validation and delivery

Run scripts/validate.ps1 or scripts/validate.sh with project-first external storage. Check formatting, tests, vet, module verification, target builds and applicable advisories. A new layout needs a rendered synthetic example and visual inspection. ZIP/XML tests establish structure and content, not visual rendering in every Word version.

Use an issue, a changelog-bearing PR and bounded independent review. Audit public source and commit metadata before copying to the public distribution repository. Keep immutable releases and record exact qualification limits. Synchronize canonical main and verify removal of issue-owned worktrees before closure.
