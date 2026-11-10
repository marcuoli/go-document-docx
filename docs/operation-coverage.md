# Document operation coverage

The operation inventory accounts for the active reference helpers and legacy wrappers. Private source paths, hashes and original method names remain in the owning source issue's evidence. This public matrix describes reusable capabilities without importing application identities or content.

| Operation | Go API or composition | Contract |
| --- | --- | --- |
| New/reset document state | document.New; editing.New/Open | Per-document state and caption counters replace global resets. |
| Core properties and contextual values | editing.SetMetadata; Native.Metadata; report.ScopedTitle; template values | Caller-authored identity, labels and dates. Naming/version/language policy remains outside. |
| Paragraph text, repetition and blank lines | editing.Paragraph/Normal/Paragraph.Append; ordinary Go loops | Literal UTF-8; CRLF normalizes to one soft break; no implicit translation. |
| Font/effects/alignment/spacing/indent | ParagraphOptions, TextStyle, Paragraph.Native | Distances in twips, sizes in half-points; nil effects inherit, explicit false/zero override. |
| Headings and progress | editing.Heading; HeadingProgress | Levels 1–9, explicit page breaks, black default headings and per-editor notifications. |
| Styled body topics and code | ParagraphOptions; editing.Code | Separate body formatting from navigable headings. Code remains literal. |
| Lists and numbered styles | editing.List | Native numbering, separate bullet/decimal instances, levels 0–8. |
| Captions and object inventories | editing.Caption; caller-owned inventory | Counters start at one; table/figure labels come from the caller. |
| Bookmarks and internal/external links | Paragraph.Bookmark/Link; Table.Link | Unique names and valid targets; external web/mail relationships are never fetched. |
| Horizontal rule and box | Paragraph.Rule | Explicit borders and neutral fill. |
| Contents and page fields | editing.Contents; native field API; existing footer | Word updates field results/page numbers; generation does not paginate. |
| Page dimensions/style/break | document.Options; editing.SetPageLayout/PageBreak; Native sections | Preserve section-local budgets and gutters; no lazy header creation during traversal. |
| Derived styles | editing.ParagraphStyle | New named style definitions are retained with loaded templates. Existing template-style edits through Native follow upstream limitations. |
| Matrix/key-value tables | editing.Table; tabledata.Data/KeyValues | Owned rectangular data; empty data stays empty instead of an invented measurement row. |
| Column selection, transpose and limits | tabledata.Select/Column/Transpose/Limit; HTML Select/SelectNames | Preserve order and identity; duplicate/missing selections fail. |
| Table sorting and display formats | tabledata.Sort/Value; editing.Column.Format | Stable exact numbers, explicit date layouts and binary units; no global locale or coercion of blanks to zero. |
| Grid/preferred/physical-cell widths | tabledata.AvailableWidth/Widths; editing.Table | cm/mm/in/pt/percent, positive widths and proportional scaling under the section budget. |
| Table/cell/row margins | TableOptions.Margins; Table.SetMargins/CellMargins/RowMargins | Explicit twips, including zero. |
| Row/column/cell font and colors | Table.RowStyle/ColumnStyle/CellStyle | Explicit styles/fills with meaningful false/black overrides. |
| Horizontal/vertical alignment and row size | Table.Align/Height; Native cells/rows | Left/top override inheritance; exact heights are opt-in and can clip content. |
| Repeating headers and row splitting | Table.HeaderRows; Native row controls | Standard row-level flags, including explicit false. |
| Single-line cells | editing.Column.SingleLine | No-wrap with fit-text explicitly disabled to avoid character stretching. |
| Display shortening and full-value access | tabledata.Shorten; editing.Column.Ellipsis | Unicode-aware and opt-in; full text remains in a ScreenTip. Print/PDF consumers need their own unshortened evidence. |
| HTML file/table selection and composition | htmlcontent.ReadTable/AppendTable/Append | Bounded reader input; exact ID/summary/column identity, explicit headerless schemas and RGB metadata. |
| Images and inline images | Paragraph.Image; editing.AddImage | Caller-provided PNG/JPEG/GIF bytes, dimensions in twips, preserved aspect ratio when one dimension is zero. |
| Text/image template rendering | template.Render | Strict literal placeholders and whole-paragraph image placeholders, including represented cells/headers/footers. |
| Closing-page template selection/composition | Caller selection plus template.Compose | Main owns page geometry/header/footer/theme/metadata; copied body/resources have remapped identities. |
| Font embedding and font selection | fonts.Embed; caller asset resolver | Complete caller-provided faces, technical flags, obfuscated parts and valid relationships; no bundled/discovered fonts. |
| Saving/checksum/registration | publication.Publish/PublishSource/Save/FromBytes | Atomic staging/replacement, immutable byte snapshots, installed state, byte count and SHA-512. Registration stays caller-owned. |
| Ordered action summaries and navigation | report/actions.Add; caller sorting/filtering | Opaque risk/impact/priority labels, caller order/visibility and optional fills/bookmarks. No inferred action or clean pass. |
| Source-specific diagnostics, evidence notices and cleanup | Caller | Provider parsing, incomplete-assessment policy, asset/customer lookup, ORM registration, retention and deletion are not document mechanisms. |
| Legacy layout wrappers | Existing primitives and caller configuration | No application-specific templates, notices, slide objects or asset paths are copied. |

## Explicit input limitations

Template rendering supports literal text keys and whole-paragraph image keys. It rejects executable Jinja controls/filters, unsupported field/revision/attribute locations, missing keys and layouts skipped by the official replacement API. A PAGE field immediately adjacent to a placeholder can be such a rejected layout; generation preserves the original input and returns an error. It does not silently remove the field.

HTML import supports block prose, inline emphasis/code/links, lists, caller-resolved images and rectangular tables. Nested tables, row/column spans, CSS layout and custom list starts are not a general HTML renderer. Headerless rows require a caller-supplied schema. Unsupported nonempty color metadata fails instead of being silently discarded.

Closing-page composition is deliberately bounded: no multi-section closing bodies, notes/comments, altChunks, tracked changes, picture bullets or reference fields requiring rewrite. Internal hyperlink bookmarks and ordinary lists/images are supported. Main document headers/footers/theme and metadata remain authoritative.

Helper link/style amendments require represented simple layouts. Field-bearing table cells, helper-styled runs after native fields, and shortened/linked merged or complex cells fail explicitly. Use supported native operations for other layouts and qualify their serialized behavior. The preserved-template style adapter guarantees helper-owned new definitions; it does not claim every mutation of an existing native style is retained.

Font collections, subsets and bitmap-only/restricted embedding are unsupported. Package-level synthetic tests verify key/relationship structure; installed-font and reader behavior remains an independent visual qualification.
