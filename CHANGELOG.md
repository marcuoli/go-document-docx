# Changelog

## Unreleased

- Export finalized DOCX sources as PDF with desktop Word, preserving headings as PDF bookmarks and using atomic publication (#48).

## 0.10.0

- Add literal ParagraphRuns with per-run RGB color and paragraph-wide bookmarks. Existing paragraph methods retain their signatures and output (#46).

## 0.9.1

- Refresh the official Go dependency graph with x/net v0.63.0, x/tools v0.53.0, go-cmp v0.7.0 and Goldmark v1.8.7. Use govulncheck v1.10.0 for validation. Keep the Go 1.27.4 shipping requirement and verify the separate Go 1.27.3 source boundary (#44).

## 0.9.0

- Reject case-variant bookmark duplicates before Word can replace a destination; reserve tooltip/imported names ignoring case and remap imported case-alias anchors (#42).

- Add literal Cell.Bookmark/Cell.LinkTarget destinations and styled internal links, plus layout-free Document.BookmarkNext anchors; preflight names/targets, preserve tooltip-only output and reuse editor validation/styles (#42).
- Use keyed external Cell literals for the additive navigation fields. Verify forward references, template collisions and native Word field-refresh/navigation without changing dependencies (#42).

## 0.8.0

- Record the pre-existing native Word qualification limit for minimal templates with official docxgo v2.16.0; retain the released dependency and track upstream resolution separately (#40).

- Bound repeated template expansion before replacement; preserve local XML namespace bindings and inherited compound formatting; keep independent HTML lists separate without changing shared List sequences (#39).
- Add owned NewList/Item sequences for explicit numbering restarts, strengthen OPC admission regressions, and remove redundant table-data copies while preserving ownership and ordering (#39).

## 0.7.1

- Use the standard slices.Index helper for native paragraph/table lookup, preserving first-match and missing-node behavior without changing public APIs or dependencies (#35).

## 0.7.0

- Require keyed external Table and CellRun literals when adopting the new optional fields; preserve existing keyed defaults.

- Reverify Go 1.27.4 and all selected module/tool releases against current official sources; retain the current graph and separately requalify source compatibility (#36).

- Add exact-style SetParagraphSpaceBefore formatting with explicit twip spacing, preserved package/content properties and conservative inherited line-unit conflict rejection (#34).

- Add optional literal Cell.ScreenTip hover text through internal self-anchor hyperlinks, preserving run appearance and empty-option output without external relationships (#33).

- Add optional literal CellRun.Italic emphasis while preserving inherited defaults, colors, line breaks and caller-owned formatting policy (#32).

- Add an optional half-point FontSize override for individual literal tables, preserving inherited typography and existing output by default (#31).

- Use black text on white for literal code-table headers, preserving white monospace code on black, exact content and ordinary data-table appearances (#30).

## 0.6.0

- Add explicit literal Cell.Alignment paragraph overrides for text and rich runs, preserving inherited defaults and code-table formatting without interpreting values.

## 0.5.0

- Add owned literal Cell.Runs spans with independent RGB colors and inherited cell emphasis, preserving the existing Cell.Text path and table engine; reject ambiguous content before output. Cell now contains a slice, requiring keyed literals and field-aware comparisons.

## 0.4.1

- Require and validate Go 1.27.4, refresh all selected upstream module versions and the vulnerability scanner, and distinguish independently tested source compatibility from the shipping module/build requirement.

## 0.4.0

- Add literal CodeTable blocks to the small document API: repeated bold headers, white Consolas text on black, natural wrapping and splittable body rows; preserve existing table styles and text/publication limits.

## 0.3.0

- Keep Go source checkouts in LF format so the Windows formatter gate matches committed source and Linux CI.

- Add opt-in ComposeWithOptions boundary-spacing trimming to prevent empty tail paragraphs from creating blank pages; preserve the existing Compose API/defaults and explicit content/breaks.

- Add optional Windows desktop Word TOC/caption-index refresh as a publication.Source decorator with isolated staging, macro-disabled opening, exact error propagation and explicit native coverage; normal rendering stays portable.

- Add HeadingOnNewPage with a paragraph-bound page break and omit a preceding table spacer at that boundary, avoiding blank pages after full tables while retaining existing Heading/PageBreak behavior.

- Keep appended-document pagination on the imported paragraph instead of a standalone break run that can create an empty page after a full table.

- Add style-selected paragraph alignment for retained document packages, preserving literal text, fields and unchanged parts; demonstrate justified prose in a separate paragraph-alignment example.

- Add validated literal table appearance options with owned settings and backward-compatible nil rendering; use a compact high-contrast grid in technical-report defaults while preserving cell colors, fill tones and exact text.

- Add optional validated RGB text colors and bold emphasis for literal document cells, preserving inherited styles and existing fill tones.
- Add literal purpose paragraphs with keep-next/keep-lines hints so table introductions can remain with their data.

## 0.2.2

- Prevent composed Word style collisions by isolating imported names/aliases as custom styles and retaining property values in schema order.
- Emit ordered emphasis, explicit shading patterns and terminal numbering cleanup identifiers; retain only the main document's note relationships.
- Reject imported table-of-contents fields rather than leaving their style/bookmark references bound to another document.

## 0.2.1

- Add layout-preserving inline literal template rendering with split-run substitution, unchanged-part retention and explicit unsupported-location checks.

## 0.2.0

- Add opt-in general authoring, explicit formatting, native navigation/lists/fields, section-aware tables, images and reusable action summaries while preserving the small report API.
- Add exact table transformations and display controls, bounded HTML/literal-template import, closing-page composition and caller-supplied font packaging.
- Add generic atomic document sources, immutable serialized snapshots and committed size/SHA-512 receipts.
- Document operation coverage, caller responsibilities, upstream limitations and comment/ownership contracts; validate both runnable examples.

## 0.1.0

- Add native DOCX document composition with configurable A4 report defaults, literal text, navigable headings, explicit table widths, repeated headers, page fields and soft-break compatibility.
- Add caller-authored findings and shared atomic publication with cancellation and overwrite handling.
