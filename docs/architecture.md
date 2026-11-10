# Architecture

The document package owns literal-text composition and the small generated OOXML vocabulary. The report package supplies configurable technical-report defaults and caller-authored finding layout. The publication package reuses go-atomicfile for inspected destinations, staging and atomic replacement. There is no command framework, database connection, customer policy or global mutable configuration.

The standard archive/zip and encoding/xml packages support the generated format. A review of the MIT-licensed godocx v0.1.5 API found that its high-level table wrapper does not expose the explicit grid widths, cell properties and repeating row headers required here. A fixed generated OOXML vocabulary avoids a second high-level dependency plus a post-processing adapter. No third-party source is copied or patched. Reassess upstream libraries if template loading, images or general document editing become requirements.

All data is literal text. Invalid UTF-8 and XML characters are rejected before output rather than silently dropped. Newlines become Word soft breaks and tabs become Word tabs. Table widths and font sizes have explicit units. Cell tones only select fills; the caller supplies statuses, comparisons, recommendations and evidence.

Input text is bounded to 64 MiB, table cells to one million and the generated main XML part to 128 MiB. These bounds do not establish a fixed peak-memory limit. Documents are owned mutable builders and must be used serially. Writers are never closed by document.Write. Publication may return installed=true with a later cleanup error; callers must preserve that distinction.

The initial API creates documents; it does not read arbitrary templates, parse HTML, interpret SQL, evaluate findings, install fonts, embed proprietary assets, or convert documents to PDF. Page rendering is a separate verification step. Application names and private provenance remain outside public source.
