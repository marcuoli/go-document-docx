# Table appearance qualification

Checked on 2026-11-12 with Go 1.27.3 on Windows amd64, CGO_ENABLED=0, GOWORK=off and GOFLAGS=-mod=readonly. Full `go test -count=1 ./...`, `go vet ./...` and `go mod verify` passed.

The first test build exposed an internal writer call still using its old signature. After supplying document options, a color regression exposed a whole-document assertion that counted intentional header colors as leakage. It now checks the neutral body cells directly, retaining literal escaping, exact values, explicit result colors and defensive input ownership. Both corrections received bounded independent source review with no actionable findings.

New appearance regressions cover defensive settings ownership, invalid options, historical nil-style output, cell hints and exact text. A composed integration sample passed Microsoft365 OpenXML validation with zero errors. This schema result does not establish visual or native Word acceptance.

Rendered inspection remains pending: development storage failed new writes during renderer launch, and a separate fresh file-write probe reproduced the device error. No rendered-page acceptance, native Word acceptance, release/tag publication or compiler/dependency change is claimed. Keep issue #11, its predecessor integration and worktree cleanup gates open until qualification and reviewed integration finish.


A later access check allowed the same composed sample to render successfully to 14 pages. Every page was inspected, including table headers, grid, literal values, missing readings, difference hints and a tight footer boundary. The document hash was unchanged after rendering and a fresh schema check returned zero errors. Native editor opening and cached-field refresh remain separate acceptance gates.

A subsequent optional QA PNG file write failed with OSError errno 22 (Invalid argument). Affected writes stopped; no storage repair or retry probe was attempted. The same crop encoded successfully in memory. This observation does not identify or resolve the underlying file-write fault. Original report/template content and immutable releases remain unchanged.


On 2026-11-12, a later native-editor validation opened the composed sample in Word 16.0.20527.20196 without an unreadable-content or repair dialog. Its contents/caption fields were refreshed in a separate copy, saved and closed. All 27 tables retained identical cell text; the original sample and private source assets were unchanged. The saved copy passed ZIP CRC and zero-error Microsoft365 OpenXML checks, and all 15 rendered pages were inspected. No new device/integrity error was observed during that validation window. This is qualification of that saved sample, not automatic field pagination by the library or proof that a prior storage fault is fixed. Released integration and predecessor cleanup gates remain open.
