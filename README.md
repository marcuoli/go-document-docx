# go-document-docx

Reusable Go Word documents with configurable report defaults and atomic publication.

The small report path remains available:

```go
options := report.Defaults()
options.Header = "Technical review"
doc, err := document.New(options)
if err != nil { return err }
doc.Title("Environment Configuration Review")
doc.Paragraph("Scope and conclusion supplied by the caller.")
installed, err := publication.Publish(ctx, path, false, doc)
```

Import functional packages under `github.com/marcuoli/go-document-docx`. The destination directory must exist; overwrite is explicit. Inspect installed state even when an error is returned.

For extended authoring:

```go
doc, err := editing.New(report.Defaults())
if err != nil { return err }
_, err = doc.Normal("Caller-authored prose.")
if err != nil { return err }
receipt, err := publication.Save(ctx, path, false, doc)
```

Extended packages cover styled paragraphs/headings/code, lists, captions, bookmarks/links, contents fields, tables and transformations, HTML import, images, literal templates, closing-page composition, caller-supplied fonts and saved-file receipts. The official document engine is opt-in; the basic report example retains its small import graph.

The default preset uses A4, 0.75-inch margins, Oracle Sans references, 9.5-point body text, 11/10-point headings and 8-point tables. Fonts are referenced without distributing them. Caller options/templates can change geometry, families, sizes, header/footer text and labels. Extended distances use twips and font sizes use half-points.

Runnable examples require explicit output paths:
- `go run ./examples/report <output.docx>` uses the small report API.
- `go run ./examples/authoring <output.docx>` demonstrates the extended packages with synthetic content.

Read [operation coverage and limits](docs/operation-coverage.md), [architecture](docs/architecture.md), [sharing decisions](docs/code-sharing.md), [dependencies](docs/dependencies.md) and [the comment convention](CONTRIBUTING.md). Editors are serial mutable builders. A failed authoring call can leave partial state; discard it before publication. Translation, evidence policy, application identities/assets, registration and retention stay with the caller.

Use template.RenderText for inline literal replacement while preserving an existing template package, including artwork and fields. Use template.Render for supported general-model text/image templates. See operation coverage for input limits.

## Cell text colors

Literal document cells accept an optional six-digit RGB TextColor, without a leading #. Empty preserves the inherited paragraph color. Text color and fill Tone are independent; callers choose the semantics and supply literal labels or glyphs. Invalid colors are rejected before serialization. Font availability and glyph appearance require qualification in the intended reader.

Cells can also opt into Bold emphasis. False preserves inherited emphasis; true emits direct bold before color in the run-property sequence. Literal labels, including angle brackets, remain text. The report example demonstrates colored checkmarks and bold colored placeholders; interpretation remains caller-owned.

Use ParagraphWithNext for a brief literal introduction that should stay with the following table. It keeps body typography and adds reader pagination hints without creating a heading or outline entry. A reader may still split content taller than a page.


### Literal table appearance

`report.Defaults()` selects compact 8-point technical tables with bold white header text on black, a half-point grid and centered cells. Each call owns an independent appearance value. Cell text colors and semantic difference/unavailable fills still take precedence in their respective cells.

```go
options := report.Defaults()
options.TableStyle.HeaderFill = "243746" // Optional alternate header palette.
doc, err := document.New(options)
// New owns the settings; later caller changes cannot alter doc.
```

Set `options.TableStyle = nil` to retain the literal renderer's historical appearance. `document.TableStyle` uses RGB colors and twip padding/spacing; its zero padding/spacing values are explicit. It preserves natural wrapping and adds no text-fitting or fixed-height policy. The runnable `examples/report` consumer demonstrates configurable headers and independent cell hints. See the API documentation for accepted alignment and size bounds.

## Paragraph alignment

Use formatting.AlignParagraphs(ctx, source, "justify", "", "Normal", "Lead") to justify selected prose without changing heading or data-cell styles. The empty style ID selects paragraphs without an explicit style. Matching prose inside layout tables is included; headers and footers remain untouched. Other supported alignments are left, center and right.

Style IDs are exact. Apply alignment before Compose renames imported styles, or supply the retained document's actual IDs. This targeted formatter preserves text, fields and unchanged package parts rather than loading and rewriting the complete native authoring model. The paragraph-alignment example demonstrates the publication path without changing the small report example; editing.ParagraphOptions remains available for newly authored native content.

Use the document.Document method HeadingOnNewPage(text, level) for a section that starts on a new page.
The boundary belongs to the heading, avoiding an empty break/spacer paragraph
after a full table. Existing Heading and PageBreak calls retain their behavior.

## Optional desktop Word TOC refresh

Import word and wrap an existing publication.Source with word.WithUpdatedFields(source, word.Options{}), then pass it to publication.Save. The decorator populates existing tables of contents and caption indexes, repaginates, refreshes TOC page numbers and saves before the destination is installed. It does not create a TOC or update arbitrary fields. Word may rewrite package XML, pagination and cached fields; this mode does not promise byte-identical template parts.

Windows and installed, configured desktop Microsoft Word in an interactive user profile are required. Normal authoring and publication stay portable and never invoke Word. Unsupported/unavailable hosts return word.ErrUnavailable; failures before commit preserve the destination. The wrapper creates and closes its own Word instance and never attaches to an active one. Macros are disabled for opening and the link-update preference is restored afterward. Options.TemporaryDirectory scopes owned staging; relative paths are made absolute before Word opens them.

Calls are serialized in this process. Cancellation is checked between synchronous COM calls and before publication; an in-flight Word call cannot be interrupted or assigned a hard timeout. No user Word process is force-terminated. Callers own source/writer lifetime and should serialize use.

The examples/word-refresh command accepts separate input/output paths; output publication requires a new destination. Enable the native synthetic TOC test explicitly with DOCX_WORD_INTEGRATION=1 and go test -tags=integration -run TestNativeWordPopulatesTOC ./word.

Implementation uses the official MIT-licensed github.com/go-ole/go-ole v1.3.0 release (upstream and module proxy checked 2026-11-13). It remains Windows-only and requires no CGO. Official method references checked on that date:

- https://learn.microsoft.com/en-us/office/vba/api/word.tableofcontents.update
- https://learn.microsoft.com/en-us/office/vba/api/word.document.repaginate
- https://learn.microsoft.com/en-us/office/vba/api/word.tableofcontents.updatepagenumbers
- https://learn.microsoft.com/en-us/office/vba/api/word.application.automationsecurity

Caption indexes use TOC fields too. Refreshing TablesOfFigures alongside TablesOfContents clears stale caption bookmark references; an index with no matching captions shows Word's empty-index message.

## Optional spacing at composition boundaries

Compose keeps its existing signature and retains main-document tail spacing. ComposeWithOptions accepts ComposeOptions{TrimTrailingSpacing: true} to omit trailing formatting-only paragraphs immediately before the imported-page boundary. This prevents empty padding paragraphs from spilling onto a blank page after a variable-length contents page. Explicit page/section properties, runs, bookmarks, drawings and unrecognized content are retained. Inherited paragraph-style pagination is not inspected; opt in only when the main template's tail is layout padding. Neither source byte slice is changed.