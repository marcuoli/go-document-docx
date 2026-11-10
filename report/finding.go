package report

import (
	"github.com/marcuoli/go-document-docx/document"
	"strings"
)

// Finding carries caller-authored report text. Status has no policy semantics.
type Finding struct {
	Title, Status, Observation, Recommendation, Evidence string
}

// Labels supplies the caller's language for finding fields. No global locale is used.
type Labels struct {
	Status, Observation, Recommendation, Evidence string
}

// AddFinding appends a heading and labeled paragraphs, omitting only empty values.
// It does not assign severity, infer recommendations or reinterpret observations.
func AddFinding(doc *document.Document, finding Finding, labels Labels) {
	doc.Heading(finding.Title, 3)
	// Keep status and the first observation in one paragraph so a heading cannot
	// be stranded with only its status at the bottom of a page.
	var lead []string
	for _, field := range [][2]string{{labels.Status, finding.Status}, {labels.Observation, finding.Observation}} {
		if field[1] != "" {
			lead = append(lead, field[0]+": "+field[1])
		}
	}
	if len(lead) > 0 {
		doc.Paragraph(strings.Join(lead, "\n"))
	}
	for _, field := range [][2]string{{labels.Recommendation, finding.Recommendation}, {labels.Evidence, finding.Evidence}} {
		if field[1] != "" {
			doc.Paragraph(field[0] + ": " + field[1])
		}
	}
}
