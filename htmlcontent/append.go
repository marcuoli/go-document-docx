package htmlcontent

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/marcuoli/go-document-docx/editing"
	"golang.org/x/net/html"
)

// Options controls HTML composition. Image resolves caller-approved resource
// identities to bytes; nil rejects image elements. No default network/file lookup
// exists. Tables use explicit Table options and caller-supplied layout metadata.
type Options struct {
	Table  TableOptions
	Layout editing.TableOptions
	Image  func(context.Context, string) ([]byte, error)
}

// Append imports headings, prose, inline emphasis/code/links, lists, rules,
// rectangular tables and caller-resolved images. Script/style/head content is
// ignored. CSS layout, Jinja and document-specific evidence rules are not inferred.
func Append(ctx context.Context, doc *editing.Document, reader io.Reader, options Options) error {
	root, err := parse(ctx, reader)
	if err != nil {
		return err
	}
	var block func(*html.Node, int, bool) error
	block = func(n *html.Node, level int, numbered bool) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if n.Type == html.TextNode {
			if strings.TrimSpace(n.Data) != "" {
				_, err := doc.Paragraph(n.Data, editing.ParagraphOptions{})
				return err
			}
			return nil
		}
		if n.Type != html.ElementNode && n.Type != html.DocumentNode {
			return nil
		}
		switch n.Data {
		case "script", "style", "head":
			return nil
		case "table":
			t, err := readTable(n, options.Table)
			if err != nil {
				return err
			}
			_, err = AppendTable(doc, t, options.Layout)
			return err
		case "h1", "h2", "h3", "h4", "h5", "h6":
			_, err := doc.Heading(strings.TrimSpace(text(n)), int(n.Data[1]-'0'), false)
			return err
		case "ul", "ol":
			if n.Data == "ol" && attr(n, "start") != "" && attr(n, "start") != "1" {
				return fmt.Errorf("HTML list start requires an explicit native numbering definition")
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.Data == "li" {
					if err := block(c, level, n.Data == "ol"); err != nil {
						return err
					}
				}
			}
			return nil
		case "li":
			p, err := doc.List("", numbered, level)
			if err != nil {
				return err
			}
			if err := inline(ctx, p, n, editing.TextStyle{}, options); err != nil {
				return err
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && (c.Data == "ul" || c.Data == "ol") {
					if err := block(c, level+1, false); err != nil {
						return err
					}
				}
			}
			return nil
		case "p", "pre", "blockquote":
			p, err := doc.Paragraph("", editing.ParagraphOptions{})
			if err != nil {
				return err
			}
			if id := attr(n, "id"); id != "" {
				if err := p.Bookmark(id); err != nil {
					return err
				}
			}
			style := editing.TextStyle{}
			if n.Data == "pre" {
				style.Font = "Consolas"
			}
			return inline(ctx, p, n, style, options)
		case "br":
			_, err := doc.Paragraph("", editing.ParagraphOptions{})
			return err
		case "hr":
			p, err := doc.Paragraph("", editing.ParagraphOptions{})
			if err != nil {
				return err
			}
			return p.Rule(false)
		case "img":
			p, err := doc.Paragraph("", editing.ParagraphOptions{})
			if err != nil {
				return err
			}
			return appendImage(ctx, p, n, options)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err := block(c, level, numbered); err != nil {
				return err
			}
		}
		return nil
	}
	return block(root, 0, false)
}
func inline(ctx context.Context, p *editing.Paragraph, n *html.Node, style editing.TextStyle, options Options) error {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.Type == html.TextNode {
			if err := p.Append(c.Data, style); err != nil {
				return err
			}
			continue
		}
		if c.Type != html.ElementNode {
			continue
		}
		next := style
		switch c.Data {
		case "script", "style", "ul", "ol":
			continue
		case "strong", "b":
			next.Bold = new(true)
		case "em", "i":
			next.Italic = new(true)
		case "u":
			next.Underline = new(true)
		case "s", "strike", "del":
			next.Strike = new(true)
		case "code":
			next.Font = "Consolas"
		case "br":
			if err := p.Append("\n", style); err != nil {
				return err
			}
			continue
		case "a":
			target := attr(c, "href")
			if target == "" {
				if err := inline(ctx, p, c, next, options); err != nil {
					return err
				}
				continue
			}
			internal := strings.HasPrefix(target, "#")
			if internal {
				target = strings.TrimPrefix(target, "#")
			}
			if err := p.Link(text(c), target, attr(c, "title"), internal); err != nil {
				return err
			}
			continue
		case "img":
			if err := appendImage(ctx, p, c, options); err != nil {
				return err
			}
			continue
		}
		if err := inline(ctx, p, c, next, options); err != nil {
			return err
		}
	}
	return nil
}
func appendImage(ctx context.Context, p *editing.Paragraph, n *html.Node, options Options) error {
	if options.Image == nil {
		return fmt.Errorf("HTML image requires a caller resolver")
	}
	data, err := options.Image(ctx, attr(n, "src"))
	if err != nil {
		return err
	}
	dimensions := [2]int{}
	for i, name := range []string{"width", "height"} {
		if value := attr(n, name); value != "" {
			pixels, err := strconv.Atoi(value)
			if err != nil || pixels < 1 || pixels > 2112 {
				return fmt.Errorf("invalid HTML image dimensions")
			}
			dimensions[i] = pixels * 15
		}
	}
	return p.Image(data, dimensions[0], dimensions[1])
}
