// Package actions composes caller-authored action summaries with optional navigation.
// It owns presentation only; ranking, filtering and acceptance stay with the caller.
package actions

import (
	"fmt"
	"github.com/marcuoli/go-document-docx/editing"
	"github.com/marcuoli/go-document-docx/tabledata"
	"strings"
)

// Entry carries caller-authored summary text and optional navigation/color.
// Risk, Impact and Priority are opaque labels; this package does not evaluate,
// filter, rank or infer actions. Callers supply the intended order and visibility.
type Entry struct{ Section, Text, Risk, Impact, Priority, Bookmark, Color string }

// Labels supplies the caller's language for the five summary columns.
type Labels struct{ Section, Action, Risk, Impact, Priority string }

// Add appends the caller's ordered summary and links explicit bookmarks.
// Empty actions produce only headers; absence is never interpreted as a clean pass.
// Use tabledata.Sort or an application comparator before rendering when required.
func Add(doc *editing.Document, actions []Entry, labels Labels, options editing.TableOptions) (*editing.Table, error) {
	data := tabledata.Data{Headers: []string{labels.Section, labels.Action, labels.Risk, labels.Impact, labels.Priority}}
	for _, header := range data.Headers {
		if strings.TrimSpace(header) == "" {
			return nil, fmt.Errorf("action labels are required")
		}
	}
	for _, a := range actions {
		data.Rows = append(data.Rows, []string{a.Section, a.Text, a.Risk, a.Impact, a.Priority})
	}
	table, err := doc.Table(data, options)
	if err != nil {
		return nil, err
	}
	for i, action := range actions {
		if action.Color != "" {
			if err := table.RowStyle(i+1, editing.TextStyle{}, action.Color); err != nil {
				return nil, err
			}
		}
		if action.Bookmark != "" {
			if err := table.Link(i+1, 0, action.Section, action.Bookmark, action.Section, true); err != nil {
				return nil, err
			}
		}
	}
	return table, nil
}
