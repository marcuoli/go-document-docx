# Selected paragraph spacing qualification

`formatting.SetParagraphSpaceBefore` sets a direct before-spacing property on
exact selected style IDs in the main document, including matching table/textbox
paragraphs. Empty style ID selects paragraphs with no explicit style. The SDK
accepts 0..31680 twips (1/1440 inch); 1..256 style IDs are required. It does not
insert spacer paragraphs, change templates or interpret a number of blank lines.

Automatic-before spacing is explicitly disabled on selected paragraphs. The API
conservatively rejects beforeLines in selected direct properties or anywhere in
package style/numbering definitions. This includes inactive/unused definitions;
it does not attempt to resolve conditional style/numbering inheritance. Apply it
to an admitted generated body before composing templates when that is the desired
scope. No-match returns an exact independent input copy without inspecting those
definitions. Other spacing properties and reader page-top/adjacent-spacing
behavior remain unchanged.

## Validation and acceptance

Source checks on 2026-11-16 used Go 1.27.4, Windows amd64, CGO disabled, GOWORK off
and readonly dependencies. Focused new/existing alignment regressions, full source
tests and vet passed. Tests verify ownership, exact selection, zero/upper bounds,
repeat/no-match byte identity, alias namespace updates, preserved foreign
attributes, XML property order, fields/bookmarks/fonts/pagination hints and
unchanged package parts. Invalid options, malformed input, spacing conflicts and
cancellation return nil; even a later conflict after an earlier selected paragraph
cannot mutate caller input or return partial output.

The runnable example is `examples/paragraph-spacing`. The 2026-11-17 release
review passed configured tests/race/vet/module/build and source/binary scans.
Final native page-start/layout/TOC validation belongs to the consumer; numeric
property tests do not claim a fixed number of blank lines. Release receipts
record immutable publication, canonical synchronization and source cleanup.

Microsoft references checked 2026-11-16:

- [Automatic-before spacing](https://learn.microsoft.com/en-us/dotnet/api/documentformat.openxml.wordprocessing.spacingbetweenlines.beforeautospacing?view=openxml-3.0.1)
- [Line-unit before spacing](https://learn.microsoft.com/en-us/dotnet/api/documentformat.openxml.wordprocessing.spacingbetweenlines.beforelines?view=openxml-3.0.1)
