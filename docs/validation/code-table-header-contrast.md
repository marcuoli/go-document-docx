# Code table header contrast

Literal code-table headers use black text on white. Code bodies remain white
Consolas on black. This default CodeTable palette is independent of ordinary
data-table appearance. Font size, literal source text, repeated headers,
body splitting and existing limits are unchanged.

## Validation and acceptance

On 2026-11-15, Go 1.27.4 with official readonly modules passed go test ./...
and go vet ./.... Focused tests assert both row palettes and dedicated style
colors, literal whitespace and Unicode, repeated headers and split-body
behavior. Ordinary table controls retain their previous appearance.

The existing code-table example rendered to 15 pages with LibreOffice 26.8.2.
Visual review covered every page, including continued headers and the final
section. Package checks confirmed both code tables use black-on-white headers
and white-on-black bodies. This is synthetic renderer qualification; it does
not replace native-reader visual qualification or guarantee pagination in
other readers. Fonts remain referenced rather than embedded.

The 2026-11-17 release review passed full configured tests/race/vet/module/build
and source/binary scans and independent source review. The example remains
byte-identical to this inspected 15-page artifact, so its visual qualification
is reused without broadening the reader claim. Immutable tag/archive/module,
canonical synchronization and exact source-cleanup receipts accompany delivery.
