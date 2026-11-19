# Literal internal navigation

Issue #42 adds named destinations and internal links to the existing literal
document path. Generic rendering remains independent of the extended editor.

## Contract and compatibility

`Cell.Bookmark string` anchors that cell's literal paragraph.
`Cell.LinkTarget string` links its literal text/runs to a named destination.
`(*Document).BookmarkNext(name string)` queues a destination on the next authored
paragraph/heading without adding a paragraph. Multiple queued names share that
paragraph; tables/code tables and page breaks leave the queue pending.

Names are 1–40 ASCII letters/digits/underscores, with a letter or underscore
first. Names duplicated ignoring ASCII case are rejected because Word otherwise
replaces a destination. Serialized names retain spelling; authored targets must
match it exactly. Imported case-alias anchors are remapped to the same destination.
Forward references are accepted.
Write rejects invalid/duplicate names, dangling queued requests and unresolved
targets before writing any ZIP bytes. User names and generated tooltip names
receive unique numeric IDs; generated names avoid all reserved user names.

Styled navigation uses the same default color 315357 and underline as editing
links; explicit Cell/CellRun colors remain authoritative. Existing bold/italic,
font sizes, alignment, breaks, whitespace, Unicode and table appearances remain.
Navigation plus ScreenTip is one hyperlink. Tooltip-only self-links keep their
previous appearance and empty options preserve the old output.

Use keyed external Cell literals for these additional fields. There is no new
module dependency, third-party modification, external target fetching or
application formatting policy.

## Qualification

Go 1.27.4 regression tests cover forward and backward cell/paragraph destinations,
rich text, queued anchors across tables/breaks, duplicate/invalid/missing targets,
one wrapper, generated-name collisions, ownership, cancellation and repeated
serialization. Composition tests include colliding IDs/names and 40-character
names whose imported names require bounded remapping, even when caller names
already occupy prefixed/generated candidates; links retain their exact
intended destination rather than jumping to the original document's bookmark.

The two-page synthetic example was rendered with LibreOffice 26.8.2 and visually
inspected: both navigation labels are underlined, and the destination text,
rich-run colors, line break and table boundaries remain legible without clipping.
This is a separate reader check from native Word navigation.

On 19 November 2026, the opt-in Windows integration test using desktop Word 16.0
published a composed synthetic document through word.WithUpdatedFields, opened
the result in a separately owned hidden Word instance and invoked Follow on both
links. Each selection started at its exact named paragraph, including the
renamed colliding imported bookmark. The test creates no external links and
restores the Word settings it changes. Native hover/UI gestures, arbitrary
readers, custom templates and final caller layouts remain separate qualification.

The pre-existing official-backend minimal-template limitation in issue #40 is
unchanged. This literal rendering/composition qualification does not repair or
qualify that distinct editing.New/minimal Open/Render path.
