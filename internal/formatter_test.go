package internal

import (
	"strings"
	"testing"
)

func TestFormatTelegramHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// === Basic formatting ===
		{
			name:     "bold",
			input:    "**bold text**",
			expected: "<b>bold text</b>",
		},
		{
			name:     "italic",
			input:    "*italic text*",
			expected: "<i>italic text</i>",
		},
		{
			name:     "strikethrough",
			input:    "~~deleted~~",
			expected: "<s>deleted</s>",
		},
		{
			name:     "inline code",
			input:    "`some code`",
			expected: "<code>some code</code>",
		},

		// === Nested formatting ===
		{
			name:     "bold inside italic",
			input:    "*italic **bold** italic*",
			expected: "<i>italic <b>bold</b> italic</i>",
		},
		{
			name:     "italic inside bold",
			input:    "**bold *italic* bold**",
			expected: "<b>bold <i>italic</i> bold</b>",
		},
		{
			name:     "strikethrough inside bold",
			input:    "**bold ~~strike~~ bold**",
			expected: "<b>bold <s>strike</s> bold</b>",
		},
		{
			name:     "triple nesting",
			input:    "**bold *italic ~~strike~~***",
			expected: "<b>bold <i>italic <s>strike</s></i></b>",
		},

		// === Code blocks ===
		{
			name:     "fenced code block without language",
			input:    "```\nfoo bar\n```",
			expected: "<pre>foo bar</pre>",
		},
		{
			name:     "fenced code block with language",
			input:    "```python\nprint('hello')\n```",
			expected: `<pre><code class="language-python">print('hello')</code></pre>`,
		},
		{
			name:     "code block with markdown inside",
			input:    "```\n**not bold**\n```",
			expected: "<pre>**not bold**</pre>",
		},
		{
			name:     "code block with html inside",
			input:    "```\n<div>hello</div>\n```",
			expected: "<pre>&lt;div&gt;hello&lt;/div&gt;</pre>",
		},
		{
			name:     "code block with ampersand",
			input:    "```\na & b\n```",
			expected: "<pre>a &amp; b</pre>",
		},
		{
			name:     "multiline code block",
			input:    "```go\nfunc main() {\n    fmt.Println(\"hi\")\n}\n```",
			expected: "<pre><code class=\"language-go\">func main() {\n    fmt.Println(\"hi\")\n}</code></pre>",
		},

		// === Links ===
		{
			name:     "basic link",
			input:    "[click here](https://example.com)",
			expected: `<a href="https://example.com">click here</a>`,
		},
		{
			name:     "link with special chars in url",
			input:    "[search](https://example.com/q?a=1&b=2)",
			expected: `<a href="https://example.com/q?a=1&amp;b=2">search</a>`,
		},
		{
			name:     "bold link text",
			input:    "[**bold link**](https://example.com)",
			expected: `<a href="https://example.com"><b>bold link</b></a>`,
		},

		// === Images ===
		{
			name:     "image with alt text",
			input:    "![screenshot](https://example.com/img.png)",
			expected: `<a href="https://example.com/img.png">screenshot</a>`,
		},
		{
			name:     "image without alt text",
			input:    "![](https://example.com/img.png)",
			expected: `<a href="https://example.com/img.png">https://example.com/img.png</a>`,
		},

		// === Blockquotes ===
		{
			name:     "simple blockquote",
			input:    "> quoted text",
			expected: "<blockquote>quoted text</blockquote>",
		},
		{
			name:     "multiline blockquote",
			input:    "> line one\n> line two",
			expected: "<blockquote>line one\nline two</blockquote>",
		},
		{
			name:     "blockquote with formatting",
			input:    "> **bold** and *italic*",
			expected: "<blockquote><b>bold</b> and <i>italic</i></blockquote>",
		},
		{
			name:     "nested blockquote",
			input:    "> outer\n>> inner",
			expected: "<blockquote>outer\n\n<blockquote>inner</blockquote></blockquote>",
		},

		// === Headings ===
		{
			name:     "h1",
			input:    "# Title",
			expected: "<b>Title</b>",
		},
		{
			name:     "h3",
			input:    "### Subsection",
			expected: "<b>Subsection</b>",
		},
		{
			name:     "heading with inline formatting",
			input:    "## **Bold** heading",
			expected: "<b><b>Bold</b> heading</b>",
		},

		// === Lists ===
		{
			name:     "unordered list",
			input:    "- one\n- two\n- three",
			expected: "• one\n• two\n• three",
		},
		{
			name:     "nested unordered list",
			input:    "- top\n  - nested\n  - nested2\n- top2",
			expected: "• top\n    ◦ nested\n    ◦ nested2\n• top2",
		},
		{
			name:     "ordered list",
			input:    "1. first\n2. second\n3. third",
			expected: "1. first\n2. second\n3. third",
		},
		{
			name:     "nested ordered list",
			input:    "1. outer one\n   1. inner one\n   2. inner two\n2. outer two",
			expected: "1. outer one\n    1. inner one\n    2. inner two\n2. outer two",
		},
		{
			name:     "mixed nested list",
			input:    "1. ordered\n   - unordered nested\n   - unordered nested2\n2. ordered again",
			expected: "1. ordered\n    ◦ unordered nested\n    ◦ unordered nested2\n2. ordered again",
		},
		{
			name:     "task list",
			input:    "- [ ] todo\n- [x] done",
			expected: "⬜ todo\n✅ done",
		},

		// === Tables ===
		{
			name:  "basic table",
			input: "| Name | Age |\n|------|-----|\n| Alice | 30 |\n| Bob | 25 |",
			expected: "<pre>" +
				"Name  | Age\n" +
				"------+----\n" +
				"Alice | 30\n" +
				"Bob   | 25" +
				"</pre>",
		},

		// === Escaping ===
		{
			name:     "escape angle brackets",
			input:    "Use <div> for layout",
			expected: "Use &lt;div&gt; for layout",
		},
		{
			name:     "escape ampersand",
			input:    "A & B",
			expected: "A &amp; B",
		},

		// === Thematic break ===
		{
			name:     "thematic break",
			input:    "before\n\n---\n\nafter",
			expected: "before\n\n———\n\nafter",
		},

		// === Edge cases ===
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text",
			input:    "just some text",
			expected: "just some text",
		},
		{
			name:     "multiple blank lines collapse",
			input:    "paragraph one\n\n\n\n\nparagraph two",
			expected: "paragraph one\n\nparagraph two",
		},
		{
			name:     "soft line break preserved",
			input:    "line one\nline two",
			expected: "line one\nline two",
		},

		// === Escaping inside code spans ===
		{
			name:     "html tags inside inline code",
			input:    "The `<div>` element and `</script>` tag",
			expected: "The <code>&lt;div&gt;</code> element and <code>&lt;/script&gt;</code> tag",
		},
		{
			name:     "ampersand inside inline code",
			input:    "Use `&amp;` for ampersand",
			expected: "Use <code>&amp;amp;</code> for ampersand",
		},
		{
			name:     "angle brackets and ampersands in inline code",
			input:    "Check `x > 0 && y < 10` first",
			expected: "Check <code>x &gt; 0 &amp;&amp; y &lt; 10</code> first",
		},

		// === Real-world LLM response ===
		{
			name: "llm explanation with code",
			input: "Here's how to fix the bug:\n\n" +
				"1. Open the config file\n" +
				"2. Change the `timeout` value\n" +
				"3. Restart the service\n\n" +
				"```python\ntimeout = 30\n```\n\n" +
				"This should resolve the issue.",
			expected: "Here's how to fix the bug:\n\n" +
				"1. Open the config file\n" +
				"2. Change the <code>timeout</code> value\n" +
				"3. Restart the service\n\n" +
				"<pre><code class=\"language-python\">timeout = 30</code></pre>\n\n" +
				"This should resolve the issue.",
		},

		// === Comprehensive complex case ===
		{
			name: "complex mixed content with all features",
			input: "## API Migration Guide\n\n" +
				"Here's what changed in v2 & why you should care about <breaking> changes:\n\n" +
				"1. The `Config` struct now uses `map[string]any` instead of `map[string]interface{}`\n" +
				"2. **All** handlers must return `(T, error)` where `T` satisfies `fmt.Stringer`\n" +
				"3. The `<Context>` type was renamed to `RequestCtx`\n\n" +
				"- Top-level item with **bold** and *italic* and `code` and ~~strike~~\n" +
				"  - Nested item referencing `fmt.Errorf(\"value: %v\", err)`\n" +
				"  - Another nested with [a link](https://example.com/path?q=1&r=2)\n" +
				"    - Deeply nested: entities like `<div>`, `&amp;`, and `</script>`\n" +
				"  - Back to second level\n" +
				"- Second top-level\n\n" +
				"> **Important:** When migrating, ensure all `<T>` type params are updated.\n" +
				"> See the [migration docs](https://docs.example.com) for details.\n\n" +
				"- [ ] Update `go.mod` dependencies\n" +
				"- [x] Replace `interface{}` with `any`\n" +
				"- [ ] Test with `go test -race ./...`\n\n" +
				"---\n\n" +
				"| Symbol | HTML Entity | Description |\n" +
				"|--------|-------------|-------------|\n" +
				"| <      | &lt;        | less than   |\n" +
				"| >      | &gt;        | greater than|\n" +
				"| &      | &amp;       | ampersand   |\n\n" +
				"```go\n" +
				"func Handle[T fmt.Stringer](ctx *RequestCtx, fn func() (T, error)) {\n" +
				"    result, err := fn()\n" +
				"    if err != nil {\n" +
				"        ctx.Error(fmt.Sprintf(\"<error>: %s\", err))\n" +
				"        return\n" +
				"    }\n" +
				"    ctx.JSON(map[string]any{\"data\": result.String()})\n" +
				"}\n" +
				"```\n\n" +
				"Finally, check `config.Get(\"timeout\") > 0 && config.Get(\"retries\") < 10` before deploying.",
			expected: "<b>API Migration Guide</b>\n\n" +
				"Here's what changed in v2 &amp; why you should care about &lt;breaking&gt; changes:\n\n" +
				"1. The <code>Config</code> struct now uses <code>map[string]any</code> instead of <code>map[string]interface{}</code>\n" +
				"2. <b>All</b> handlers must return <code>(T, error)</code> where <code>T</code> satisfies <code>fmt.Stringer</code>\n" +
				"3. The <code>&lt;Context&gt;</code> type was renamed to <code>RequestCtx</code>\n\n" +
				"• Top-level item with <b>bold</b> and <i>italic</i> and <code>code</code> and <s>strike</s>\n" +
				"    ◦ Nested item referencing <code>fmt.Errorf(\"value: %v\", err)</code>\n" +
				"    ◦ Another nested with <a href=\"https://example.com/path?q=1&amp;r=2\">a link</a>\n" +
				"        ◦ Deeply nested: entities like <code>&lt;div&gt;</code>, <code>&amp;amp;</code>, and <code>&lt;/script&gt;</code>\n" +
				"    ◦ Back to second level\n" +
				"• Second top-level\n\n" +
				"<blockquote><b>Important:</b> When migrating, ensure all <code>&lt;T&gt;</code> type params are updated.\n" +
				"See the <a href=\"https://docs.example.com\">migration docs</a> for details.</blockquote>\n\n" +
				"⬜ Update <code>go.mod</code> dependencies\n" +
				"✅ Replace <code>interface{}</code> with <code>any</code>\n" +
				"⬜ Test with <code>go test -race ./...</code>\n\n" +
				"———\n\n" +
				"<pre>" +
				"Symbol | HTML Entity | Description\n" +
				"-------+-------------+-------------\n" +
				"&lt;      | &amp;lt;        | less than\n" +
				"&gt;      | &amp;gt;        | greater than\n" +
				"&amp;      | &amp;amp;       | ampersand" +
				"</pre>\n\n" +
				"<pre><code class=\"language-go\">" +
				"func Handle[T fmt.Stringer](ctx *RequestCtx, fn func() (T, error)) {\n" +
				"    result, err := fn()\n" +
				"    if err != nil {\n" +
				"        ctx.Error(fmt.Sprintf(\"&lt;error&gt;: %s\", err))\n" +
				"        return\n" +
				"    }\n" +
				"    ctx.JSON(map[string]any{\"data\": result.String()})\n" +
				"}" +
				"</code></pre>\n\n" +
				"Finally, check <code>config.Get(\"timeout\") &gt; 0 &amp;&amp; config.Get(\"retries\") &lt; 10</code> before deploying.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTelegramHTML(tt.input)
			if got != tt.expected {
				t.Errorf("FormatTelegramHTML(%q)\ngot:      %q\nexpected: %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSplitMessageWithHTMLBoundary(t *testing.T) {
	// Document known limitation: splitMessage can split inside HTML tags.
	// The plain-text fallback in SendMessage handles this gracefully.
	content := "<pre>" + strings.Repeat("x", 5000) + "</pre>"
	chunks := splitMessage(content)
	if len(chunks) < 2 {
		t.Fatal("expected message to be split into multiple chunks")
	}
	// First chunk will have unclosed <pre> - this is the documented limitation.
	// In production, Telegram's HTML parser rejects it and SendMessage retries as plain text.
	if !strings.HasPrefix(chunks[0], "<pre>") {
		t.Error("first chunk should start with <pre>")
	}
}
