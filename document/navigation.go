package document

import (
	"context"
	"fmt"
	"strings"

	"github.com/marcuoli/go-document-docx/internal/navigation"
)

// BookmarkNext anchors the next authored paragraph or heading without adding
// layout content. Tables, code tables and page breaks do not consume the request;
// multiple requests share that next paragraph. Names use 1..40 ASCII letters,
// digits or underscores, beginning with a letter or underscore. Write rejects
// invalid/duplicate names and requests left without a paragraph. Names and link
// targets use exact spelling. Like other construction methods, use serially.
func (d *Document) BookmarkNext(name string) {
	d.pendingBookmarks = append(d.pendingBookmarks, name)
}

func (d *Document) appendParagraph(item block) {
	item.bookmarks = d.pendingBookmarks
	d.pendingBookmarks = nil
	d.blocks = append(d.blocks, item)
}

// Resolve all names before body generation so forward references are ordinary
// links, and no invalid navigation can leave a partially written ZIP.
func (d *Document) navigationNames(ctx context.Context) (map[string]bool, error) {
	if len(d.pendingBookmarks) > 0 {
		return nil, fmt.Errorf("bookmark request has no following paragraph")
	}
	names := map[string]bool{}
	add := func(name string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := navigation.ValidateBookmarkName(name); err != nil {
			return err
		}
		if names[name] {
			return fmt.Errorf("duplicate bookmark %q", name)
		}
		names[name] = true
		return nil
	}
	for _, item := range d.blocks {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, name := range item.bookmarks {
			if err := add(name); err != nil {
				return nil, err
			}
		}
		if item.table == nil {
			continue
		}
		for _, row := range item.table.Rows {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			for _, cell := range row {
				if cell.Bookmark != "" {
					if err := add(cell.Bookmark); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	for _, item := range d.blocks {
		if item.table == nil {
			continue
		}
		for _, row := range item.table.Rows {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			for _, cell := range row {
				if cell.LinkTarget == "" {
					continue
				}
				if err := navigation.ValidateBookmarkName(cell.LinkTarget); err != nil {
					return nil, err
				}
				if !names[cell.LinkTarget] {
					return nil, fmt.Errorf("missing bookmark target %q", cell.LinkTarget)
				}
			}
		}
	}
	return names, nil
}

type bookmarkState struct {
	id       int
	reserved map[string]bool
}

func (s *bookmarkState) next(ctx context.Context, selfLink bool) (int, error) {
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		s.id++
		if !selfLink || !s.reserved[fmt.Sprintf("DocxCellTip%d", s.id)] {
			return s.id, nil
		}
	}
}
func writeBookmarkStart(ctx context.Context, b *strings.Builder, name string, id int) error {
	return writeBookmarkMarkup(ctx, b, fmt.Sprintf(`<w:bookmarkStart w:id="%d" w:name="%s"/>`, id, name))
}
func writeBookmarkEnd(ctx context.Context, b *strings.Builder, id int) error {
	return writeBookmarkMarkup(ctx, b, fmt.Sprintf(`<w:bookmarkEnd w:id="%d"/>`, id))
}

// Validated names are at most 40 ASCII bytes, so this small temporary markup is
// bounded. Check before every marker: one paragraph may carry many bookmarks.
func writeBookmarkMarkup(ctx context.Context, b *strings.Builder, markup string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(markup) > (128<<20)-b.Len() {
		return fmt.Errorf("document XML exceeds 128 MiB")
	}
	b.WriteString(markup)
	return nil
}
