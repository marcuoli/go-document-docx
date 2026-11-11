# Preserved literal template rendering

**Goal:** Render inline literal values while keeping existing package artwork, fields, sections and assets.
**Architecture:** Reuse bounded OPC and etree operations for a focused text splice. Retain the existing general Render/image API. Application assets, identity and policy remain outside the module.
**Compatibility:** Go 1.27.3, CGO disabled for artifacts, GOWORK off, released dependencies without replacements.
**Tracking:** Source issue 5.

- [x] Add RenderText with reverse split-run splicing, first-character formatting, unchanged-part retention, explicit unsupported locations and bounded/cancelled output.
- [x] Test Unicode/once-only values, artwork/field/header/nested paragraph preservation, malformed input, limits and cancellation using synthetic fixtures.
- [ ] Complete independent review and configured Windows/Linux/race/advisory checks.
- [ ] Publish audited v0.2.1, verify release/source/module parity, sync canonical main and remove exact issue worktree before closure.
