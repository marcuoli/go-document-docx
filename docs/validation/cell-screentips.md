# Literal cell ScreenTip qualification

`document.Cell.ScreenTip` adds caller-authored hover text to the whole literal
cell paragraph. Empty preserves existing bytes. Nonempty text wraps the existing
runs in an internal self-anchor hyperlink with a generated paired bookmark;
no external relationship or Hyperlink run style is added. Visible content and
formatting remain caller-owned. Link interaction and hover display depend on
reader settings; this is not an alternate-text or arbitrary-reader guarantee.

## Validation and acceptance

Source qualification on 2026-11-16 used Go 1.27.4, Windows amd64, CGO disabled,
GOWORK off and readonly official dependencies. Focused/full source tests and vet
passed. Tests check exact tooltip escaping, owned snapshots, unique paired
bookmarks across tables, matching self-anchors, default output, all unchanged
package parts and unchanged styled runs after removing only tooltip wrappers.
Colors, italic, bold, table size, alignment, CRLF/tabs/Unicode, cancellation and
writer errors are covered. Text/expanded-XML overflows reject before output.

Independent review exposed unbounded temporary attribute expansion and delayed
cancellation. Regressions reproduced both. UTF-8-safe chunked escaping now writes
through a context-aware XML budget writer, without a fully expanded temporary
string. Expansion and deterministic cancellation regressions cover the fix.

The existing isolated desktop Word post-processing path opened and saved a
synthetic document and retained both tooltip strings/internal targets, literal
glyph text and explicit color/italic properties without introducing link styling.
The opt-in `TestNativeWordRetainsCellScreenTips` checks this behavior. Actual hover
interaction and final template-composed layout require separate consumer UI QA;
XML retention does not establish hover behavior or every reader's appearance.
Word may normalize an internal link into a HYPERLINK field with `\l` for the
internal target and `\o` for the description. Audits of a composed/saved document
must accept equivalent native representations rather than require a particular
`w:hyperlink` count; Word's Hyperlink.ScreenTip property can confirm its value.

Bookmark IDs and names are document-local generated implementation details and
must not be retained as a caller API. Each standalone document restarts allocation;
composing documents must remap both bookmark IDs/names and hyperlink anchors. The
shared template.Compose path provides that mapping. Final consumer composition
and Word refresh must verify target uniqueness and resolution with its own inputs.

The 2026-11-17 configured release checks and follow-up independent review pass.
Final tag/archive/module/canonical/source-cleanup receipts are separate delivery
evidence. Consumer hover/UI and final template-layout acceptance retain the
limits above; publishing the SDK does not establish those additional claims.
