# Changelog

## Unreleased

- Keep appended-document pagination on the imported paragraph instead of a standalone break run that can create an empty page after a full table.

- Add style-selected paragraph alignment for retained document packages, preserving literal text, fields and unchanged parts; demonstrate justified prose in a separate paragraph-alignment example.

- Add validated literal table appearance options with owned settings and backward-compatible nil rendering; use a compact high-contrast grid in technical-report defaults while preserving cell colors, fill tones and exact text.

## 0.3.0

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
