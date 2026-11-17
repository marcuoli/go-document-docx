# Literal code table qualification

The additive `document.Document.CodeTable(header, code string)` API uses a
repeating bold header and one splittable body cell. The header uses black Consolas
text on white; the body uses white Consolas text on black at the configured
table size. This palette is explicit and
independent of ordinary table appearance. Existing data-table and paragraph
styles remain unchanged, including when the same document contains both.

## Historical reader qualification

The reader checks below qualified the previous all-dark header palette. They
remain historical evidence for literal content and pagination. Current header
contrast checks are recorded in [header contrast](code-table-header-contrast.md).

Qualification on 2026-11-15 used Go 1.27.3, Windows amd64, CGO disabled,
GOWORK off and readonly official modules. Full tests, vet, module verification,
library/example builds, the small-report dependency guard and source/binary
scans passed. Race tests also passed with CGO temporarily enabled. The
documented required-module-only GO-2026-6859 exception remains; no affected
imported package or called symbol was reported. This is a bounded scanner
result, not a universal absence-of-vulnerabilities claim.

Focused tests verify literal whitespace, CRLF line-break normalization, tabs,
empty lines, XML escaping, Unicode including supplementary characters, long
unbroken text beyond 32,767 characters, exact style selection, repeated-header
and splittable-body flags, invalid text and aggregate-limit rejection before
output, cancellation, unchanged existing table XML and existing style definitions.
Serialization reuses the existing bounded context-aware literal writer.

The runnable `examples/code-table` sample was rendered with Microsoft Word
16.0.20527 and LibreOffice 26.8.2 on Windows. Both produced 15 pages. All pages
were visually inspected for clipping, long-line wrapping, header repetition,
page splits and the explicit following-section boundary. Native Word retained
both code cells exactly after normalizing its paragraph/soft-break markers,
including the 40,000-character body. Native PDF text verification found all
500 numbered source comments in order and a repeated header on every page
containing them. Word opened the generated package normally; this qualification
does not claim a new full OpenXML schema-validator run.

The synthetic Unicode sample rendered accented Latin, Japanese and a
supplementary emoji using the installed reader/font fallback. Fonts are
referenced, not embedded. Other fonts, readers, language shaping, arbitrary
documents and different pagination configurations require their own checks.

The code block is an in-memory literal formatter, not a database CLOB reader or
source-completeness policy. Existing whole-document limits remain 64 MiB of
aggregate text and 128 MiB of expanded document XML. Exceeding a bound fails
instead of returning a complete-looking prefix. CRLF and CR become visual line
breaks; tabs remain tabs and their visual width is reader-dependent. Callers own
retrieval, localization, source budgets, availability and atomic publication.

Independent review found no actionable code defect. Its test-quality suggestion
was applied: cell style IDs and each code style's font, size, color, emphasis
and pagination properties are asserted directly. Release receipts separately
record exact CI, integration, tag/archive verification and source cleanup.
