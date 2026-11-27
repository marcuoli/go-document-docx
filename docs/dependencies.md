# Released dependency assessment

## Current build selection

Refreshed on 2026-11-27 against the official Go feed, release history and
upstream stable tags. The module, validation scripts and CI require Go 1.27.4.
Validation uses the standalone govulncheck v1.10.0.

The direct library releases are etree v1.8.1, go-ole v1.3.0, go-atomicfile
v0.1.1, docxgo/v2 v2.16.0, okage v0.23.0, x/image v0.48.0, and x/net v0.63.0.
The SVG renderer update passed the existing artwork admission and rendering
tests. The docxgo provenance and minimal-template qualification below still apply.

The selected graph also uses x/crypto v0.59.0, x/mod v0.43.0, x/sync v0.25.0,
x/sys v0.50.0, x/term v0.48.0, x/text v0.44.0, x/tools v0.53.0,
go-cmp v0.7.0 and Goldmark v1.8.7. x/tools imports Goldmark's v1 paths, so
adding the separately versioned v2 module would leave v1 selected. Telemetry
has no stable tag. The graph retains the revision selected by x/tools v0.53.0,
v0.0.0-20261116191925-e77a43154acb. The typesetting test utilities also have
no stable tag and remain at their upstream-selected revision. The manifest
deliberately retains selected graph-only constraints.

Formatting, tests, race checks, vet, module verification, and example builds
passed on Windows and Linux amd64 with Go 1.27.4. The configured source and
example-binary vulnerability scans reported no findings on either platform.

## Historical qualification

Checked on 2026-11-10 for Go 1.27.3. Final compiler/graph/target and advisory results are retained with validation evidence.

| Module | Selected release | Responsibility and license |
| --- | --- | --- |
| github.com/marcuoli/go-atomicfile | v0.1.1 | First-party atomic publication |
| github.com/mmonterroca/docxgo/v2 | v2.16.0 | General Word model, images, metadata, sections, serialization and supported text replacement; declared MIT |
| github.com/go-ole/go-ole | v1.3.0 | Optional Windows COM automation through the official upstream release; MIT |
| github.com/beevik/etree | v1.8.1 | Namespace-preserving XML tree operations; BSD 2-Clause |
| golang.org/x/net | v0.62.0 | HTML parser; BSD 3-Clause |

The selected document library's current LICENSE declares MIT and its upstream provenance audit explains historical AGPL snapshots and the rewritten releases. This records the upstream determination rather than an independent legal assessment. No historical AGPL-only snapshot or local dependency fork is used.

Official sources: [docxgo release source](https://github.com/mmonterroca/docxgo/tree/v2.16.0), [license and provenance](https://github.com/mmonterroca/docxgo/blob/v2.16.0/docs/PROVENANCE_AUDIT.md), [etree](https://github.com/beevik/etree/tree/v1.8.1), [Go HTML parser](https://pkg.go.dev/golang.org/x/net/html).

The library's general MergeTemplate operation can clear an adjacent empty field-bearing run. The supported literal ReplaceText operation is used with skipped-layout receipts; unsupported field-adjacent templates return an explicit error. Its retained-template style part also requires new helper-owned definitions to be serialized separately through the official writer and appended to the artifact. Neither adaptation modifies dependency source or claims those upstream limitations are fixed.

Other candidates were assessed before implementation. The narrower high-level table wrappers did not cover the report controls; a licensed runtime was not selected. The broader requirements now justify the opt-in general engine while the small report path keeps its original import surface. Review newly released compatible fixes before the next dependency/build qualification.

All dependencies are consumed as official releases with signed module sums. No local replacement, fork, vendored modification, cache patch or build-time rewriting is part of final validation.

Qualification on 2026-11-11 retains the required Go 1.27.3 toolchain. govulncheck reports GO-2026-6859 (net/http MaxBytesReader header race), fixed in Go 1.27.4, at required-module level; no affected imported package or called symbol was reported for this module or either example binary. This is a bounded scanner result, not proof of exploitability or universal absence of vulnerabilities. No compiler migration is included in the Word compatibility patch. Recheck this advisory at the next configured build review.

The optional Windows-only Word package uses the official go-ole v1.3.0 release without CGO. Upstream source and license: https://github.com/go-ole/go-ole/tree/v1.3.0 . The default portable renderer does not invoke Word. This dependency is not a replacement document engine.

## Minimal-template native Word qualification

Official docxgo v2.16.0 can add theme, web-settings and document-property parts
while retaining a minimal input package's content-type manifest. Native Word
rejects that output when the added parts lack their specific declarations;
LibreOffice and generic ZIP/XML checks can still accept it. This was reproduced
on v0.7.1 and the audit-remediation candidate (#40).

Retain the official release and upstream-only policy. A diagnostic package
with declarations added is evidence only, not a production dependency workaround.
The 2026-11-27 official tag and module checks still select v2.16.0.
Recheck official releases or supported public configuration before the next
module publication, then qualify editing.New and minimal Open/Render directly
in Word without repair. Complete caller-supplied Word templates are qualified
separately; that result does not resolve the minimal-template path.
