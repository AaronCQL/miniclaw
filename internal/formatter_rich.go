package internal

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// FormatTelegramRichHTML renders Markdown as Bot API 10.1 Rich HTML.
//
// Rich HTML follows HTML whitespace rules (literal newlines collapse), so block
// separation is expressed through tags and line breaks use <br>, not "\n".
func FormatTelegramRichHTML(markdown string) string {
	if markdown == "" {
		return ""
	}

	source := []byte(markdown)
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	var buf strings.Builder

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch node := n.(type) {
		case *ast.Paragraph:
			// Tight list items use TextBlock; loose ones use Paragraph. Only
			// wrap in <p> outside list items so bullets stay on one line.
			if _, inListItem := node.Parent().(*ast.ListItem); !inListItem {
				if entering {
					buf.WriteString("<p>")
				} else {
					buf.WriteString("</p>")
				}
			}

		case *ast.TextBlock:
			// Tight list item content has no wrapper; walk its inline children.

		case *ast.Heading:
			level := node.Level
			if level < 1 {
				level = 1
			}
			if level > 6 {
				level = 6
			}
			if entering {
				fmt.Fprintf(&buf, "<h%d>", level)
			} else {
				fmt.Fprintf(&buf, "</h%d>", level)
			}

		case *ast.ThematicBreak:
			if entering {
				buf.WriteString("<hr/>")
			}

		case *ast.CodeBlock:
			if entering {
				buf.WriteString("<pre>")
				writeCodeBlockLines(&buf, node, source)
				buf.WriteString("</pre>")
			}
			return ast.WalkSkipChildren, nil

		case *ast.FencedCodeBlock:
			if entering {
				lang := string(node.Language(source))
				switch {
				case lang == "math":
					// Body is raw LaTeX, escaped only so the HTML parser passes it through.
					buf.WriteString("<tg-math-block>")
					writeCodeBlockLines(&buf, node, source)
					buf.WriteString("</tg-math-block>")
				case lang != "":
					fmt.Fprintf(&buf, `<pre><code class="language-%s">`, escapeHTML(lang))
					writeCodeBlockLines(&buf, node, source)
					buf.WriteString("</code></pre>")
				default:
					buf.WriteString("<pre>")
					writeCodeBlockLines(&buf, node, source)
					buf.WriteString("</pre>")
				}
			}
			return ast.WalkSkipChildren, nil

		case *ast.Blockquote:
			if entering {
				buf.WriteString("<blockquote>")
			} else {
				buf.WriteString("</blockquote>")
			}

		case *ast.List:
			if entering {
				if node.IsOrdered() {
					if node.Start > 1 {
						fmt.Fprintf(&buf, `<ol start="%d">`, node.Start)
					} else {
						buf.WriteString("<ol>")
					}
				} else {
					buf.WriteString("<ul>")
				}
			} else {
				if node.IsOrdered() {
					buf.WriteString("</ol>")
				} else {
					buf.WriteString("</ul>")
				}
			}

		case *ast.ListItem:
			if entering {
				buf.WriteString("<li>")
			} else {
				buf.WriteString("</li>")
			}

		case *extast.TaskCheckBox:
			if entering {
				if node.IsChecked {
					buf.WriteString(`<input type="checkbox" checked> `)
				} else {
					buf.WriteString(`<input type="checkbox"> `)
				}
			}

		case *extast.Table:
			if entering {
				renderRichTable(&buf, node, source)
			}
			return ast.WalkSkipChildren, nil

		case *ast.Text:
			if entering {
				buf.WriteString(escapeHTML(string(node.Segment.Value(source))))
				if node.SoftLineBreak() || node.HardLineBreak() {
					buf.WriteString("<br>")
				}
			}

		case *ast.String:
			if entering {
				buf.WriteString(escapeHTML(string(node.Value)))
			}

		case *ast.CodeSpan:
			if entering {
				buf.WriteString("<code>")
			} else {
				buf.WriteString("</code>")
			}

		case *ast.Emphasis:
			open, close := "<i>", "</i>"
			if node.Level == 2 {
				open, close = "<b>", "</b>"
			}
			if entering {
				buf.WriteString(open)
			} else {
				buf.WriteString(close)
			}

		case *extast.Strikethrough:
			if entering {
				buf.WriteString("<s>")
			} else {
				buf.WriteString("</s>")
			}

		case *ast.Link:
			if entering {
				fmt.Fprintf(&buf, `<a href="%s">`, escapeHTML(string(node.Destination)))
			} else {
				buf.WriteString("</a>")
			}

		case *ast.AutoLink:
			if entering {
				url := string(node.URL(source))
				label := string(node.Label(source))
				fmt.Fprintf(&buf, `<a href="%s">%s</a>`, escapeHTML(url), escapeHTML(label))
			}

		case *ast.Image:
			// Rich messages only allow media as standalone blocks, so an inline
			// image stays a link, matching the legacy formatter.
			if entering {
				dest := string(node.Destination)
				alt := extractText(node, source)
				if alt == "" {
					alt = dest
				}
				fmt.Fprintf(&buf, `<a href="%s">%s</a>`, escapeHTML(dest), escapeHTML(alt))
			}
			return ast.WalkSkipChildren, nil

		case *ast.RawHTML:
			if entering {
				for i := range node.Segments.Len() {
					seg := node.Segments.At(i)
					buf.WriteString(escapeHTML(string(seg.Value(source))))
				}
			}

		case *ast.HTMLBlock:
			if entering {
				for i := range node.Lines().Len() {
					seg := node.Lines().At(i)
					buf.WriteString(escapeHTML(string(seg.Value(source))))
				}
				if node.HasClosure() {
					buf.WriteString(escapeHTML(string(node.ClosureLine.Value(source))))
				}
			}
		}

		return ast.WalkContinue, nil
	})

	return strings.TrimSpace(buf.String())
}

// Rich message table cells may contain inline formatting only.
func renderRichTable(buf *strings.Builder, table *extast.Table, source []byte) {
	buf.WriteString("<table>")
	for row := table.FirstChild(); row != nil; row = row.NextSibling() {
		cellTag := "td"
		if _, ok := row.(*extast.TableHeader); ok {
			cellTag = "th"
		}
		buf.WriteString("<tr>")
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			fmt.Fprintf(buf, "<%s%s>", cellTag, cellAlignAttr(cell))
			renderCellInline(buf, cell, source)
			fmt.Fprintf(buf, "</%s>", cellTag)
		}
		buf.WriteString("</tr>")
	}
	buf.WriteString("</table>")
}

func cellAlignAttr(cell ast.Node) string {
	c, ok := cell.(*extast.TableCell)
	if !ok {
		return ""
	}
	switch c.Alignment {
	case extast.AlignLeft:
		return ` align="left"`
	case extast.AlignCenter:
		return ` align="center"`
	case extast.AlignRight:
		return ` align="right"`
	default:
		return ""
	}
}
