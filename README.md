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
