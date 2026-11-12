# Table appearance qualification

Checked on 2026-11-12 with Go 1.27.3 on Windows amd64, CGO_ENABLED=0, GOWORK=off and GOFLAGS=-mod=readonly. Full `go test -count=1 ./...`, `go vet ./...` and `go mod verify` passed.

The first test build exposed an internal writer call still using its old signature. After supplying document options, a color regression exposed a whole-document assertion that counted intentional header colors as leakage. It now checks the neutral body cells directly, retaining literal escaping, exact values, explicit result colors and defensive input ownership. Both corrections received bounded independent source review with no actionable findings.

New appearance regressions cover defensive settings ownership, invalid options, historical nil-style output, cell hints and exact text. A composed integration sample passed Microsoft365 OpenXML validation with zero errors. This schema result does not establish visual or native Word acceptance.

Rendered inspection remains pending: development storage failed new writes during renderer launch, and a separate fresh file-write probe reproduced the device error. No rendered-page acceptance, native Word acceptance, release/tag publication or compiler/dependency change is claimed. Keep issue #11, its predecessor integration and worktree cleanup gates open until qualification and reviewed integration finish.
