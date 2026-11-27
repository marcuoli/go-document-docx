# go-document-docx

The [logo example](examples/logo) reads bounded PNG or JPEG artwork and places
it in a table cell. The caller selects the square size and cell, and the library
preserves the image aspect ratio. Use `Logo.PNG` to retrieve a caller-owned copy
of the prepared PNG and the original pixel dimensions for another renderer.

![Module and build requirement Go 1.27.4](https://img.shields.io/badge/module%2Fbuild-Go%201.27.4-blue)
![Source compatibility Go 1.27.3](https://img.shields.io/badge/source%20compatibility-Go%201.27.3-blue)

Reusable Go Word documents with configurable report defaults and atomic publication.

## Go versions

**Module/build requirement: Go 1.27.4.** The canonical `go.mod`, validation scripts
and CI select this current stable release, checked against the official Go feed
and release history on 2026-11-27.

**Verified source compatibility with the selected dependencies: Go 1.27.3.**
An isolated copy with only its `go` directive lowered passed tests,
vet/stdversion, module verification, and builds with Go 1.27.3 on Windows
and Linux amd64. The synthetic desktop Word integration suite also passed
on Windows. The source, dependency versions, and checksums matched the
shipping build inputs.

Go 1.27.2 rejected the selected dependency graph because `go-atomicfile` v0.1.1
requires Go 1.27.3. Okage v0.23.0 declares the same requirement. Mingo v0.19.0
corroborated the Go 1.27 dependency requirement on both platforms. The
source uses the non-type `new` argument introduced in Go 1.26. Installing
the published module requires Go 1.27.4 as declared in its manifest.

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

The default preset uses A4, 0.75-inch margins, references to Arial, 9.5-point body text, 11/10-point headings and 8-point tables. Fonts are referenced without distributing them. Caller options/templates can change geometry, families, sizes, header/footer text and labels. Extended distances use twips and font sizes use half-points.

Runnable examples require explicit output paths:
- `go run ./examples/report <output.docx>` uses the small report API.
- `go run ./examples/authoring <output.docx>` demonstrates the extended packages with synthetic content.

Read [operation coverage and limits](docs/operation-coverage.md), [architecture](docs/architecture.md), [sharing decisions](docs/code-sharing.md), [dependencies](docs/dependencies.md) and [the comment convention](CONTRIBUTING.md). Editors are serial mutable builders. A failed authoring call can leave partial state; discard it before publication. Translation, evidence policy, application identities/assets, registration and retention stay with the caller.

Native Word qualification of extended editing from minimal templates remains limited by the official backend manifest issue tracked in #40; see [the dependency qualification note](docs/dependencies.md#minimal-template-native-word-qualification). Complete Word templates and the basic literal report path have separate qualification evidence.

Use template.RenderText for inline literal replacement while preserving an existing template package, including artwork and fields. Use template.Render for supported general-model text/image templates. See operation coverage for input limits.

## Literal API compatibility

The optional `Table.FontSize`, `CellRun.Italic` and `Cell.ScreenTip` fields are
additive for keyed struct literals. External unkeyed `Table` or `CellRun` literals
must be converted to keyed literals when upgrading to v0.7.0. Cell was already
slice-bearing and non-comparable; existing ownership rules remain unchanged.

## Colored runs in body paragraphs

Use ordered literal runs when part of a paragraph needs its own text color:

```go
d.ParagraphRuns(
    document.ParagraphRun{Text: "Validation passed. "},
    document.ParagraphRun{Text: "✓", TextColor: "008000"},
)
```

An empty color inherits the body style. Supply six hexadecimal RGB digits without
a leading # for a color override. The method copies its input, keeps the text
literal and appends one body paragraph, even when called with no runs. As with
`Paragraph`, `Write` checks text and document limits before sending output.
`BookmarkNext` names the whole paragraph. The existing `Paragraph` and
`ParagraphWithNext` methods retain their behavior. See the [runnable example](examples/paragraph-runs).

## Cell text colors

Literal document cells accept an optional six-digit RGB TextColor, without a leading #. Empty preserves the inherited paragraph color. Text color and fill Tone are independent; callers choose the semantics and supply literal labels or glyphs. Invalid colors are rejected before serialization. Font availability and glyph appearance require qualification in the intended reader.

Cells can also opt into Bold emphasis. False preserves inherited emphasis; true emits direct bold before color in the run-property sequence. Literal labels, including angle brackets, remain text. The report example demonstrates colored checkmarks and bold colored placeholders; interpretation remains caller-owned.

For independent colors within one literal cell paragraph, use ordered runs:

```go
cell := document.Cell{Runs: []document.CellRun{
    {Text: "1:N/A\n"},
    {Text: "2:N/A", TextColor: "B3261E"},
}}
```

Empty or nil `Runs` keeps the existing `Cell.Text` path. Nonempty `Runs` requires
empty `Text`; ambiguous input fails `Write` before output. An empty run color
inherits `Cell.TextColor` or the paragraph style, and `Cell.Bold` applies to every
run. Cell fill tones remain independent. Text is literal, including spaces,
tabs, blank lines and Unicode; CRLF/CR become visual line breaks. `AddTable`
deep-copies run slices. Colors never evaluate labels or availability. Set a run's
`Italic: true` for native italic emphasis; false preserves inherited emphasis
and existing output. Italics applies only to that literal span and does not
select labels or alter colors, cell bold, alignment or table font size.

`Cell` now contains a slice and is not comparable. Use keyed cell literals and
compare the fields/values relevant to the caller rather than using cells as map
keys or comparing whole cells with `==`. Existing text serialization is retained.
The existing document limits of 64 MiB aggregate text and 128 MiB expanded XML
also cover run content. `go run ./examples/cell-runs <output.docx>` demonstrates
the API; fonts, tab widths and pagination remain reader-dependent.

[Cell-run qualification](docs/validation/cell-runs.md) records the tested native
colors, exact legacy-output comparison and serialization boundary regressions.

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

## Grouped table headers

Set `Table.Widths` for the underlying columns. Use `Table.HeaderRows` for formatted leading rows that repeat together in Word. `Cell.ColSpan` covers adjacent columns in a header or body row. Each row must use the full grid, with one cell for each spanned group. For a single plain header row, use `Table.Headers`. The [grouped comparison example](examples/grouped-table/main.go) shows two labels spanning their subcolumns and a row with one merged value per group.

## Cell ScreenTips

Set `Cell.ScreenTip` to caller-authored hover text for a literal cell paragraph.
Nonempty text uses a generated internal bookmark and self-anchor hyperlink,
without an external URL/relationship or an added Hyperlink run style. Text and
rich runs keep their formatting; empty ScreenTip preserves existing output.
`AddTable` owns the value. UTF-8/XML validation and aggregate document text/XML
limits include ScreenTip text. Readers control hover display and link interaction;
printing does not add the tooltip to visible cell content. Callers own all label
selection, translations and accessibility policy.
See [ScreenTip qualification](docs/validation/cell-screentips.md) for limits,
Word retention checks and document-composition constraints.

## Paragraph space before

Use `formatting.SetParagraphSpaceBefore(ctx, input, 720, "Heading1")` to set
selected main-document paragraphs to 720 twips (half an inch) of direct before
spacing. Exact style selection, including empty/no-explicit-style IDs, matches
`AlignParagraphs`. Valid input is 0..31680 twips. Automatic-before is explicitly
disabled; beforeLines on a selected paragraph or anywhere in package style or
numbering definitions is conservatively rejected, without a general style
resolver. Other spacing properties, fonts, text, fields and pagination hints
remain unchanged. Readers control page-top suppression and adjacent spacing;
no spacer paragraphs are inserted. Apply before template composition when only
the generated body should be selected. `examples/paragraph-spacing` is runnable
source; no-op returns an independent byte-identical copy and errors return nil.

## Paragraph alignment

Literal table cells accept `document.Cell{Text: "1,234.50", Alignment: "right"}`.
`Alignment` supports `left`, `center`, `right` and `justify`, for both `Text` and
`Runs`. Empty emits no direct override and preserves existing style inheritance
and output. Invalid values fail `Write` before output. Alignment applies to the
whole cell paragraph, leaves headers and sibling cells alone, and never parses
or reformats values. `AddTable` owns the supplied alignment value. Code tables
retain their existing left-aligned behavior. The runnable
`examples/cell-alignment` sample demonstrates text, rich runs and inherited
defaults; final wrapping and justification remain reader-dependent.
See [cell-alignment qualification](docs/validation/cell-alignment.md) for tested
conditions and reader limits.

Use formatting.AlignParagraphs(ctx, source, "justify", "", "Normal", "Lead") to justify selected prose without changing heading or data-cell styles. The empty style ID selects paragraphs without an explicit style. Matching prose inside layout tables is included; headers and footers remain untouched. Other supported alignments are left, center and right.

Style IDs are exact. Apply alignment before Compose renames imported styles, or supply the retained document's actual IDs. This targeted formatter preserves text, fields and unchanged package parts rather than loading and rewriting the complete native authoring model. The paragraph-alignment example demonstrates the publication path without changing the small report example; editing.ParagraphOptions remains available for newly authored native content.

Use the document.Document method HeadingOnNewPage(text, level) for a section that starts on a new page.
The boundary belongs to the heading, avoiding an empty break/spacer paragraph
after a full table. Existing Heading and PageBreak calls retain their behavior.

## Literal code tables

Use `doc.CodeTable(header, code)` on a `document.Document` for caller-authored source:

```go
doc.CodeTable("SQL source", "CREATE TABLE example_items (\n\tid NUMBER\n);")
```

The full-width, single-column table has a repeated bold header and one splittable body row. The header uses black Consolas text on white; the body uses white Consolas text on black, both at the configured table font size. Code is left aligned, wraps naturally and has no fixed row height, line numbering, translation, execution or truncation. Existing table appearances and `editing.Code` remain unchanged. The header is literal caller text, including localization; it creates no heading/TOC entry. Empty header/body strings are supported.

Spaces, indentation, tabs, blank lines and valid Unicode remain literal. CRLF/CR normalize to visual line breaks. The renderer preserves long lines and text beyond 32,767 characters; font coverage and actual wrapping/pagination depend on the installed fonts and reader. `Write` rejects invalid UTF-8/XML characters and aggregate text above 64 MiB or expanded document XML above 128 MiB before writing bytes. Construction and serialization use memory; this API does not read or stream database CLOBs. Callers own source retrieval, completeness, budgets and atomic publication.

Run `go run ./examples/code-table <output.docx>` for a synthetic multipage example. [Code-table qualification](docs/validation/code-tables.md) records rendered samples separately from structural tests; no universal reader or arbitrary-size source guarantee is implied.

## Literal internal navigation

Use `Document.BookmarkNext(name)` before a paragraph or heading, or set
`Cell.Bookmark` on an exact table-cell paragraph. A `Cell.LinkTarget` names that
destination, including a later destination in the same document.

```go
d.AddTable(document.Table{
    Headers: []string{"Section"},
    Widths: []int{5000},
    Rows: [][]document.Cell{{{
        Text: "Details", LinkTarget: "detail", ScreenTip: "Jump to the detail",
    }}},
})
d.BookmarkNext("detail")
d.Heading("Details", 1)
```

Names use 1–40 ASCII letters, digits or underscores and start with a letter or
underscore. Names duplicated ignoring ASCII case are rejected; serialized names
and targets retain exact spelling. `Write` rejects invalid/duplicate names,
unresolved targets and a queued request with no following paragraph before
emitting output. Tables/code tables and page breaks do not consume a queued
request; multiple queued names can share one paragraph.

Navigation uses the editor's default link color and underline, retaining explicit
cell/run colors, emphasis, size and literal content. `ScreenTip` annotates that
same hyperlink. Tooltip-only cells keep their existing self-links and appearance.
Empty options preserve existing output. Use keyed external `Cell` literals when
adopting these additional fields.

Run `go run ./examples/internal-navigation <output.docx>` for a synthetic example.
[Navigation qualification](docs/validation/internal-navigation.md) records
composition collisions, native Word field refresh/Follow and reader limits.

## Optional desktop Word TOC refresh

Import word and wrap an existing publication.Source with word.WithUpdatedFields(source, word.Options{}), then pass it to publication.Save. The decorator populates existing tables of contents and caption indexes, repaginates, refreshes TOC page numbers and saves before the destination is installed. It does not create a TOC or update arbitrary fields. Word may rewrite package XML, pagination and cached fields; this mode does not promise byte-identical template parts.

Windows and installed, configured desktop Microsoft Word in an interactive user profile are required. Normal authoring and publication stay portable and never invoke Word. Unsupported/unavailable hosts return word.ErrUnavailable; failures before commit preserve the destination. The wrapper creates and closes its own Word instance and never attaches to an active one. Macros are disabled for opening and the link-update preference is restored afterward. Options.TemporaryDirectory scopes owned staging; relative paths are made absolute before Word opens them.

Calls are serialized in this process. Cancellation is checked between synchronous COM calls and before publication; an in-flight Word call cannot be interrupted or assigned a hard timeout. No user Word process is force-terminated. Callers own source/writer lifetime and should serialize use.

The examples/word-refresh command accepts separate input/output paths; output publication requires a new destination. Enable the native synthetic TOC test explicitly with DOCX_WORD_INTEGRATION=1 and go test -tags=integration -run TestNativeWordPopulatesTOC ./word.

### Export a finalized DOCX to PDF

Wrap a finalized DOCX source with `word.PDF(source, word.Options{})` and pass that source to `publication.Save(ctx, pdfPath, overwrite, pdf)`. The receipt identifies the committed PDF. Word reads a private staged copy, uses its own layout for all pages, and creates PDF bookmarks from headings. The SDK streams the generated PDF from disk and preserves the input. Refresh any needed TOC fields before supplying the finalized DOCX.

Word PDF export omits automatic literal cell ScreenTip links that target their own generated range. New DOCX files give these links separate hidden anchors and hidden producer markers, even in cells with an authored bookmark. Authored links and bookmark destinations remain available. Unmarked links in older DOCX files are retained.

This operation uses the same desktop Word requirements, automation lock, and cancellation semantics as field refresh. Set `Options.TemporaryDirectory` to scope both staged files. An export failure or cancellation before commit preserves an existing PDF. Run `go run ./examples/word-pdf output.pdf` on a supported Windows host to render the synthetic example.

Implementation uses the official MIT-licensed github.com/go-ole/go-ole v1.3.0 release (upstream and module proxy checked 2026-11-13). It remains Windows-only and requires no CGO. Official method references checked on that date:

- https://learn.microsoft.com/en-us/office/vba/api/word.tableofcontents.update
- https://learn.microsoft.com/en-us/office/vba/api/word.document.repaginate
- https://learn.microsoft.com/en-us/office/vba/api/word.tableofcontents.updatepagenumbers
- https://learn.microsoft.com/en-us/office/vba/api/word.application.automationsecurity

Caption indexes use TOC fields too. Refreshing TablesOfFigures alongside TablesOfContents clears stale caption bookmark references; an index with no matching captions shows Word's empty-index message.

## Optional spacing at composition boundaries

Compose keeps its existing signature and retains main-document tail spacing. ComposeWithOptions accepts ComposeOptions{TrimTrailingSpacing: true} to omit trailing formatting-only paragraphs immediately before the imported-page boundary. This prevents empty padding paragraphs from spilling onto a blank page after a variable-length contents page. Explicit page/section properties, runs, bookmarks, drawings and unrecognized content are retained. Inherited paragraph-style pagination is not inspected; opt in only when the main template's tail is layout padding. Neither source byte slice is changed.

## Per-table font size

Literal document.Table accepts FontSize in half-points; 14 means 7 points.
Zero inherits the document's table styles. A nonzero value must be 12 through
144 and applies to this table's header and body runs, including rich spans and
complex-script text. Colors, emphasis, alignment and literal content remain
independent. Other tables, code blocks and document-wide styles are unchanged.
Write rejects invalid sizes before emitting bytes. Explicit sizes survive
supported template composition; fonts and actual wrapping remain reader-dependent.
