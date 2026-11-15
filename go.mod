module github.com/marcuoli/go-document-docx

go 1.27.4

require (
	github.com/beevik/etree v1.8.1
	github.com/go-ole/go-ole v1.3.0
	github.com/marcuoli/go-atomicfile v0.1.1
	github.com/mmonterroca/docxgo/v2 v2.16.0
	golang.org/x/net v0.62.0
)

// Keep the complete upstream module graph current, including unimported paths.
// Tidy alone removes these graph-only constraints inherited through x/net.
require (
	golang.org/x/crypto v0.59.0 // indirect
	golang.org/x/mod v0.43.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/term v0.48.0 // indirect
	golang.org/x/text v0.44.0 // indirect
)
