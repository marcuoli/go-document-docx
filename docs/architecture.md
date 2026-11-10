# Architecture

The small `document`, `report` and `publication` packages retain the established literal-report API. The basic report example does not import the extended authoring engine. Extended consumers opt into `editing`, `tabledata`, `htmlcontent`, `template`, `fonts`, and `report/actions`.

## Responsibilities

- `editing` owns a serial editor around the official docxgo API. Its Native handles expose supported upstream operations. Helpers add explicit paragraph/run formatting, headings and progress notifications, native lists/contents fields, unique bookmarks/links, captions, images, table geometry and cell/row controls.
- `tabledata` owns rectangular value snapshots and explicit selection, transpose, limits, stable typed sorting, units, exact display conversion and shortening. It preserves original spelling during sorting and never maps unavailable data to zero.
- `htmlcontent` uses x/net/html for bounded prose/list/link/image and rectangular-table import. Callers resolve image bytes and choose headerless schemas, column identity, null tokens and formatting.
- `template` uses supported literal replacement APIs, strict location receipts and collision-checked opaque tokens. Substituted values are not interpreted again. Closing-page composition imports the separate body and remaps its styles, numbering, bookmarks and transitive resources while the main document retains page geometry, headers, footers, theme and metadata.
- `fonts` packages caller-provided complete TTF/OTF faces with OpenType obfuscation, valid keys/relationships and technical embedding-flag checks. It performs no installed-font discovery.
- `publication` uses the released go-atomicfile operations for inspected destinations, staging, sync and replacement. Save returns installed state, committed byte count and SHA-512; applications own registration and retention.
- `internal/opc` uses standard ZIP/XML admission plus etree for namespace-preserving, schema-ordered document-artifact changes. `internal/nativecontent` provides read-only traversal without creating missing headers or footers.

## Dependency decisions

The initial generated-report vocabulary remains small and unchanged. The broader authoring requirements justify docxgo v2.16.0 rather than another general object model. etree v1.8.1 handles XML trees and x/net v0.62.0 supplies the HTML parser. All are official released modules; no dependency source is copied, patched, rewritten or replaced.

Targeted artifact controls supply features absent from the general API: fixed table grids/physical-cell widths, explicit left/top/black/false overrides, disabled fit-text with no-wrap, soft-break compatibility, unique named bookmarks, numbering and caller-owned new styles in retained templates. New style definitions are serialized through a fresh official upstream document and appended to the retained style part. Existing template definitions are preserved.

The selected upstream template-merge API can clear an adjacent empty field-bearing run. The supported ReplaceText API is used instead, and its skipped-layout receipt is enforced. Field-adjacent placeholders requiring unsupported replacement fail explicitly. This is an input limitation, not a dependency patch.

## Ownership and bounds

Editors and their Native handles share serial mutable ownership. A failed authoring call can leave a partial editor; discard it on error. Input values/bytes used by immutable helpers and serialized publication snapshots are copied. Native edits remain subject to the upstream API and the documented helper-layout restrictions.

The extended package envelope admits at most 64 MiB compressed, 256 MiB uncompressed, 128 MiB per part, and 4096 parts. XML admission permits depth 128, two million elements per part and 256 attributes per element. Output uses the same compressed bound. These are admission limits, not a fixed peak-memory guarantee or a full OOXML schema validator.

New document text is limited to 64 MiB and represented cells to one million. Numeric spelling is limited to 4096 bytes and scientific-exponent magnitude 4096 before big-number expansion. HTML input is UTF-8 and limited to 8 MiB. Template substitution permits 4096 occurrences. Individual fonts/images are limited to 32 MiB; decoded image dimensions are bounded to 100 million pixels.

Cancellation is checked between phases, during traversal and at writer boundaries. Third-party CPU-only serialization/parsing is synchronous. Publication never closes a caller-owned writer. A successful install can precede a cleanup error, so callers must inspect Installed even when an error is returned.

## Caller responsibilities

Translation, evidence interpretation, customer identities/assets, scope, action visibility/ranking, source-file discovery, naming/version/language conventions, database registration and retention remain caller-owned. No application identities, customer content, logos, font binaries, legal boilerplate, command framework, SQL engine or global locale/state are bundled.

Word computes pagination and field results. Font availability and the reader determine substitution. Visual qualification of a generated example does not establish every Word version, font or arbitrary template combination. See operation coverage for explicit unsupported inputs.
