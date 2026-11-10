# go-document-docx

Reusable native Go Word documents with configurable report defaults and atomic file publication.

Implementation and examples are organized by responsibility. Public consumers use functional package imports.

```go
options := report.Defaults()
options.Header = "Technical review"
doc, err := document.New(options)
if err != nil { return err }
doc.Title("Environment Configuration Review")
doc.Paragraph("Review scope and conclusion supplied by the caller.")
installed, err := publication.Publish(ctx, path, false, doc)
```

Import `github.com/marcuoli/go-document-docx/document`, `/report` and `/publication`. The destination directory must already exist; overwrite is explicit. Inspect `installed` even when an error is returned. `document.Write` also accepts an ordinary writer.

The report preset uses A4 pages, 0.75-inch margins, Oracle Sans body text at 9.5 points, 11/10-point headings and 8-point tables. It references font families without distributing font files. Change the returned options to use other fonts, sizes, dimensions, header/footer text and page labels. Titles and headings are black. Tables repeat their header, retain exact text and support neutral, difference and unavailable fills.

Run the synthetic consumer with an explicit output: `go run ./examples/report <output.docx>`. See [architecture](docs/architecture.md) and [contributing](CONTRIBUTING.md) for API limits and the comment convention. No product content, logos or legal notices are bundled.
