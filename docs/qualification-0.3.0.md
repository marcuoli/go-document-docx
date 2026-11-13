# Release 0.3.0 qualification

This additive release includes literal table text colors/emphasis and appearance,
purpose paragraphs, retained style-selected paragraph alignment, heading/composed
page boundaries, optional composition-tail spacing trim and optional Windows
desktop Word TOC/caption-index refresh. Existing default behavior remains covered
by focused compatibility tests.

## Validation and acceptance

- Required Go 1.27.3, Windows amd64, CGO disabled, GOWORK=off and readonly official
  modules: the complete owning validation script passes full tests, vet, module
  verification, library/example builds and the small-report dependency guard.
- Race tests pass with CGO temporarily enabled and the existing native compiler.
- Source and both example binary scans report no affected imported package or
  called symbol. The required-module-only GO-2026-6859 finding remains visible in
  dependencies.md; it is fixed in Go 1.27.4. This bounded result does not establish
  exploitability or universal absence of vulnerabilities.
- A scoped LF checkout rule resolves the Windows CRLF formatter mismatch.
  Every one of the 64 normalized Go files matches its committed blob exactly;
  no Go semantic change is part of that correction.
- The table sample opened in Word 16.0.20527.20196 without unreadable-content or
  repair dialogs. A separately saved/refreshed copy retained identical text in
  all 27 tables, passed ZIP CRC and zero-error Microsoft365 OpenXML validation,
  and all 15 rendered pages were inspected.
- Retained paragraph alignment passed exact-literal/field/part ownership and
  namespace-prefix regressions; its synthetic rendered sample was inspected.
- Final optional field-refresh/composition-boundary consumer qualification at
  e720ba16b98cd1d1ba3303a7f702435895a33ae5 records 13 inspected native Word pages,
  six matching page references and zero OpenXML errors. Synthetic native Word
  testing also populated the TOC and cleared stale caption-index entries.
- Bounded independent source reviews cover formatting/ownership, retained XML,
  Word lifecycle and cancellation, page boundaries and conservative tail trimming.
  Release review found no code/API-ownership or public-source privacy blockers.
- All new source commits use public noreply author and committer identities.
  No private source template or downstream application identity is distributed.

These observations qualify the tested samples and configurations, not every
document or reader. Word automation requires configured desktop Word on Windows;
cancellation remains cooperative between synchronous COM calls. The owning
release receipt records final CI, immutable tag/assets, public mirror/canonical
synchronization and exact source cleanup separately from these source checks.

See validation/table-appearance-qualification.md and
validation/paragraph-alignment.md for focused evidence and historical storage
limitations; no storage repair is claimed.
