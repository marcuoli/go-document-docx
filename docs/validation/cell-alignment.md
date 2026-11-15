# Literal cell alignment qualification

`document.Cell.Alignment` accepts empty, `left`, `center`, `right` and `justify`.
Empty keeps the existing paragraph style and serialization. Explicit alignment
applies to the one paragraph containing either `Text` or `Runs`; justification
uses OOXML `w:jc` value `both`. Values remain caller-authored literal text.

Qualification on 2026-11-15 used Go 1.27.4, Windows amd64, CGO disabled,
GOWORK off and readonly official dependencies. Focused and full tests, vet,
module verification, library/example builds and source/binary vulnerability
scans passed. Race tests passed with CGO enabled. Scans reported no vulnerabilities
for these configurations. No dependencies or build requirements changed.

Tests check every allowed value with text and rich runs, exact numeric-looking
literal text, retained colors/emphasis, ownership after caller mutation,
header/sibling isolation and rejection of unsupported/injected values before
writer output. Existing run, limit, cancellation and code-table tests pass.
Independent review found no actionable defects. The existing cell-runs example
produced byte-identical DOCX output to the native-qualified v0.5.0 baseline.

The runnable `examples/cell-alignment` sample rendered as one inspected page in
LibreOffice 26.8.2: the explicit value is right aligned, rich runs are centered,
and code retains left alignment and indentation. Justified final-line behavior
remains reader-controlled. Fonts are referenced rather than embedded. No fresh
desktop Word visual qualification or arbitrary-reader guarantee is claimed.

Release receipts separately record exact compiler/source compatibility checks,
CI, reviewed integration, immutable tags, archives, module resolution, released
example parity and temporary-source cleanup.
