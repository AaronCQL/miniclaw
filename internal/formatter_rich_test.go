package internal

import "testing"

func TestFormatTelegramRichHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// === Inline (shared with legacy HTML) ===
		{
			name:     "bold",
			input:    "**bold text**",
			expected: "<p><b>bold text</b></p>",
		},
		{
			name:     "inline code",
			input:    "`x`",
			expected: "<p><code>x</code></p>",
		},
		{
			name:     "strikethrough requires double tilde",
			input:    "~~gone~~ and ~kept~",
			expected: "<p><s>gone</s> and ~kept~</p>",
		},
		{
			name:     "link",
			input:    "[site](https://t.me/)",
			expected: `<p><a href="https://t.me/">site</a></p>`,
		},
		{
			name:     "dollar amounts stay literal text",
			input:    "$15K/mth cash + ~$1.2M vest",
			expected: "<p>$15K/mth cash + ~$1.2M vest</p>",
		},

		// === Structural upgrades over legacy ===
		{
			name:     "heading becomes h-tag not bold",
			input:    "## Q1 Report",
			expected: "<h2>Q1 Report</h2>",
		},
		{
			name:     "heading level clamps at 6",
			input:    "####### deep",
			expected: "<p>####### deep</p>", // 7 hashes is not a heading in GFM
		},
		{
			name:     "thematic break",
			input:    "---",
			expected: "<hr/>",
		},
		{
			name:     "unordered list",
			input:    "- one\n- two",
			expected: "<ul><li>one</li><li>two</li></ul>",
		},
		{
			name:     "ordered list with start",
			input:    "3. three\n4. four",
			expected: `<ol start="3"><li>three</li><li>four</li></ol>`,
		},
		{
			name:     "task list checkbox",
			input:    "- [x] done\n- [ ] todo",
			expected: `<ul><li><input type="checkbox" checked> done</li><li><input type="checkbox"> todo</li></ul>`,
		},

		// === Tables ===
		{
			name:     "table with alignment",
			input:    "| Metric | Value |\n|:-------|------:|\n| Speed | 42 |",
			expected: `<table><tr><th align="left">Metric</th><th align="right">Value</th></tr><tr><td align="left">Speed</td><td align="right">42</td></tr></table>`,
		},
		{
			name:     "table cell keeps inline formatting",
			input:    "| A |\n|---|\n| **b** |",
			expected: "<table><tr><th>A</th></tr><tr><td><b>b</b></td></tr></table>",
		},

		// === Code and maths ===
		{
			name:     "fenced code with language",
			input:    "```python\nprint(1)\n```",
			expected: "<pre><code class=\"language-python\">print(1)</code></pre>",
		},
		{
			name:     "math fence becomes math block",
			input:    "```math\nE = mc^2\n```",
			expected: "<tg-math-block>E = mc^2</tg-math-block>",
		},
		{
			name:     "html is escaped not interpreted",
			input:    "use <b> carefully",
			expected: "<p>use &lt;b&gt; carefully</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTelegramRichHTML(tt.input)
			if got != tt.expected {
				t.Errorf("FormatTelegramRichHTML(%q)\n  got:  %q\n  want: %q", tt.input, got, tt.expected)
			}
		})
	}
}
