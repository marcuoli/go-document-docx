# Literal cell run qualification

`document.Cell.Runs` adds ordered `CellRun{Text, TextColor}` spans to the existing
literal table renderer. Empty runs preserve `Cell.Text`; nonempty runs require
empty `Text`. Run color falls back to the cell color/style, cell bold applies
throughout, and fill tones remain independent. Labels never determine color.
`Cell` contains a slice and is no longer comparable; callers use keyed literals
and appropriate field/value comparisons.

## Validation and acceptance

Qualification on 2026-11-15 used Go 1.27.4, Windows amd64, CGO disabled,
GOWORK off and readonly official dependencies. Full tests, vet, module
verification, library/example builds, the small-report import guard and
source/binary advisory scans passed. Race tests passed with CGO temporarily
enabled. Current scans reported no vulnerabilities; this is a bounded result
for the tested configuration.

Focused tests cover literal mixed `1:N/A` and `2:N/A` spans, exact XML text,
Unicode, tabs, blank lines, color inheritance/overrides, all-run bold, fill
independence, deep ownership, empty runs and byte-identical equivalent legacy
cells. Invalid mixed input, RGB, UTF-8/XML and text/expanded-XML overflow fail
before writer output. Cancellation and writer error identity remain explicit.

Independent review identified two boundary cases. Regressions reproduced them
before correction: CRLF split across spans produced two breaks, and empty runs
missed render-time cancellation. Normalization now carries the CR state across
empty spans, and each run checks cancellation before serialization. Deterministic
tests verify both corrections without timing-dependent cancellation.

The runnable `examples/cell-runs` sample produced one inspected page in Word
16.0.20527 and LibreOffice 26.8.2. Native Word retained `1:N/A` and `2:N/A` on
separate literal lines in one cell; its first five characters were black and
the second line was the caller's RGB B3261E. The synthetic example also shows
bold Unicode/tabbed spans and an unchanged whole-cell color. Font fallback and
tab spacing differ between readers; fonts are referenced rather than embedded.

The existing report example produced exactly the same DOCX bytes as the
published v0.4.1 executable under the same compiler, including every legacy
table/style/paragraph part. The new API uses the same bounded literal serializer,
table layout and publication path. Existing 64 MiB aggregate-text and 128 MiB
expanded-document-XML limits remain; source retrieval, availability semantics
and document ownership stay with callers. No new full OpenXML schema-validator
or arbitrary-reader qualification is claimed.

Release receipts separately record final compiler/source-compatibility checks,
CI, immutable tag/archive/module verification, canonical synchronization and
temporary-source cleanup.
