# Code sharing decisions

| Responsibility | Existing implementation and consumers | Decision and compatibility |
| --- | --- | --- |
| Literal report generation | Existing document/report API and basic report example | Preserve its small import graph and public signatures. It remains independent of the extended authoring packages. |
| General authoring and images | Official docxgo v2.16.0; extended editor and authoring example | Use its public model, style, image, metadata, section and serialization APIs directly. Do not maintain another general document object model. |
| Literal template text | Official docxgo ReplaceText | Use strict replacement receipts and opaque intermediate tokens so values stay literal. Enforce unsupported field/layout errors rather than silently clearing existing fields. |
| XML and HTML parsing | etree v1.8.1 and x/net/html v0.62.0 | Use maintained parsers for their supported responsibilities. The small OPC adapter owns document-artifact changes and bounds; no upstream source is copied or patched. |
| Repeated package mechanics | Editor amendments, template composition and font packaging | Share bounded ZIP/XML admission, namespace handling, schema order, relative targets and relationships under internal/opc. This is private format plumbing rather than another public engine. |
| Read-only model traversal | Editor validation and template image binding | Share traversal under internal/nativecontent. Enumerate existing headers/footers without invoking lazy creation or changing inheritance. |
| Atomic publication | Released go-atomicfile/atomicfile v0.1.1; basic and extended writers | Preserve installed/cleanup semantics and destination checks. Generic Source and immutable byte snapshots avoid duplicate publication engines. |
| Exact numeric table display/sorting | Standard math/big and explicit unit/precision bounds | The existing generic comparison library was inspected; its value API compares integer/text states and does not expose the decimal sort/format responsibility here. Retain the small bounded standard-library implementation. Reassess extraction only with another concrete consumer and compatible contracts. |
| Action summaries and contextual titles | Generic presentation helpers | Accept caller-authored labels, order, navigation and colors. Do not extract business ranking, diagnostics filtering, acceptance or discovery policy. |
| Data-frame input and application storage | Caller adapters | Accept explicit rectangular values/ordered key-value entries and publication receipts. Do not introduce a frame runtime, ORM or background cleanup into a document library. |

The broader editor is opt-in. Both runnable examples validate public APIs with the same compiler and released graph. Native checks, advisory scans and rendered examples are recorded by the owning source issue; compatibility claims remain bounded to that evidence.

Layout-preserving literal text substitution belongs in template.RenderText. Existing package/namespace validation and etree editing are reused, while applications retain private template bytes and value/asset policy. Synthetic tests qualify split runs, fields, drawing nodes and unchanged parts. This opt-in path avoids reconstructing unsupported artwork through the general object model.

Simple-field cache opt-in is application policy: the module only matches explicit instructions and preserves them. Default literal rendering rejects field placeholders. This avoids treating arbitrary cached field text as durable content.

## Composed Word compatibility - issue #7

Style identity and OOXML schema order belong to the shared document/template packages. Existing etree and internal/opc primitives remain the format implementation; consumers retain template assets and layout policy. Imported styles use isolated IDs, names and aliases as custom styles, while their property values remain unchanged. Output style run properties use the schema sequence, including templates with appended overrides. Unsupported imported note references and TOC/reference fields fail explicitly; unused note parts are not duplicated. Synthetic tests cover both generation and repeated composition, singleton relationships, alias collisions, numbering cleanup tails and simple/split field admission. Native application compatibility must be verified separately from well-formed XML or alternate-renderer success.
