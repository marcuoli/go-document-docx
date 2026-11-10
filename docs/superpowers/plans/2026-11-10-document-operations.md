# Document operations implementation plan

**Goal:** Provide the reusable operations needed for complete technical document authoring, preserving the v0.1 public API.
**Architecture:** Keep the established literal report API stable. Add an editing package backed by official docxgo APIs, pure tabledata transformations, HTML parsing through x/net/html, template operations, and caller-supplied font packaging. Use etree for bounded, namespace-aware OPC adjustments that the general document API does not expose. Never modify dependency source.
**Compatibility:** Go 1.27.3, CGO disabled for artifacts, GOWORK off, released modules only. No application identities, assets, translations, ORM/retention policy, or machine paths in public source.
**Delivery:** Source-forge issue 3; public history starts at the audited v0.1.0 tag.

- [x] Inventory the active source methods and legacy-file disposition privately; publish neutral operation coverage and bounded compatibility decisions.
- [x] Add tabledata owned transforms, exact typed sorting, width units/budget, explicit formatting and Unicode display shortening. Test failures, shape/identity and exact values.
- [x] Add editing.New/Open/Native/Write, styled paragraphs/headings/lists/code/captions, links/bookmarks/TOC, table and image helpers using docxgo. Retain explicit output controls through bounded OPC adjustments. Test publication cancellation, unique identifiers, namespace/order and template preservation.
- [x] Add HTML paragraph/table import, strict template substitution and closing-page composition, caller-supplied font embedding and saved-file receipts. Reject unsupported input explicitly and preserve source bytes/destinations on failure.
- [x] Document each operation, units, ownership, concurrency and compatibility. Add a synthetic comprehensive example and render every page; preserve existing examples/API.
- [ ] Validate native tests, race, vet, module verification, compiler/build metadata and advisories. Obtain bounded independent review, correct findings, integrate, publish an audited immutable semantic release, synchronize canonical main, and remove the recorded issue worktree through Git before closure.

## Review focus

- Input references must remain exact and must not be replaced by fabricated zero/empty data.
- Template package relationships, identifiers and namespace aliases must remain valid.
- Table geometry must agree across preferred width, grid and physical cells, including spans.
- Missing resources, unsupported constructs, cancellation and failed publication must not replace a good destination.
- Embedding must use caller-provided font bytes and inspect technical embedding restrictions; no installed-font discovery or bundled assets.
