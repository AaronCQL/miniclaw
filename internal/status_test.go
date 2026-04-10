package internal

import (
	"fmt"
	"strings"
	"testing"
)

func TestStatusTracker_Empty(t *testing.T) {
	s := newStatusTracker()
	if r := s.Render(); r != "" {
		t.Errorf("Render() on empty = %q, want empty", r)
	}
	if r := s.RenderDone(); r != "" {
		t.Errorf("RenderDone() on empty = %q, want empty", r)
	}
	if r := s.RenderFinal(); r != "" {
		t.Errorf("RenderFinal() on empty = %q, want empty", r)
	}
}

func TestStatusTracker_Add_ReturnValue(t *testing.T) {
	s := newStatusTracker()

	if got := s.Add("Read", "<code>main.go</code>"); !got {
		t.Error("first Add should return true")
	}

	if got := s.Add("Bash", "<code>go test</code>"); got {
		t.Error("second Add should return false")
	}

	if got := s.Add("Bash", "<code>go test</code>"); got {
		t.Error("duplicate Add should return false")
	}

	if got := s.Add("Bash", "<code>go build</code>"); got {
		t.Error("non-first Add should return false")
	}
}

func TestStatusTracker_Add_DeduplicateConsecutive(t *testing.T) {
	s := newStatusTracker()
	s.Add("Read", "<code>a.go</code>")
	s.Add("Read", "<code>a.go</code>")
	s.Add("Read", "<code>a.go</code>")

	want := "📄 <code>a.go</code> 🟡"
	if got := s.Render(); got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestStatusTracker_Add_ExitPlanMode(t *testing.T) {
	s := newStatusTracker()

	got := s.Add("ExitPlanMode", "")
	if !got {
		t.Error("ExitPlanMode on empty tracker should return true")
	}
	if r := s.Render(); r != "" {
		t.Errorf("ExitPlanMode should not add entry, Render() = %q", r)
	}

	s.Add("Read", "file")
	if got := s.Add("ExitPlanMode", ""); got {
		t.Error("ExitPlanMode on non-empty tracker should return false")
	}
}

func TestStatusTracker_Add_TodoWriteEmptyLabel(t *testing.T) {
	s := newStatusTracker()

	got := s.Add("TodoWrite", "")
	if !got {
		t.Error("TodoWrite with empty label on empty tracker should return true")
	}
	if r := s.Render(); r != "" {
		t.Errorf("TodoWrite with empty label should not add entry, Render() = %q", r)
	}
}

func TestStatusTracker_Add_UnknownTool(t *testing.T) {
	s := newStatusTracker()
	s.Add("SomeNewTool", "doing stuff")

	want := "⚙️ doing stuff 🟡"
	if got := s.Render(); got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestStatusTracker_Add_MCPTool(t *testing.T) {
	s := newStatusTracker()
	s.Add("mcp__playwright__browser_snapshot", "MCP: Playwright Browser Snapshot")

	want := "🛠 MCP: Playwright Browser Snapshot 🟡"
	if got := s.Render(); got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestStatusTracker_Add_EmptyLabelUsesToolName(t *testing.T) {
	s := newStatusTracker()
	s.Add("Read", "")

	want := "📄 Read 🟡"
	if got := s.Render(); got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestStatusTracker_Render(t *testing.T) {
	s := newStatusTracker()
	s.Add("Read", "<code>main.go</code>")
	s.Add("Bash", "<code>go test</code>")
	s.Add("WebSearch", "golang errors")

	want := "📄 <code>main.go</code>\n⚡ <code>go test</code>\n🌐 golang errors 🟡"
	if got := s.Render(); got != want {
		t.Errorf("Render() =\n%s\nwant:\n%s", got, want)
	}
}

func TestStatusTracker_RenderDone(t *testing.T) {
	s := newStatusTracker()
	s.Add("Read", "<code>main.go</code>")
	s.Add("Bash", "<code>go test</code>")

	want := "📄 <code>main.go</code>\n⚡ <code>go test</code>"
	if got := s.RenderDone(); got != want {
		t.Errorf("RenderDone() =\n%q\nwant:\n%q", got, want)
	}
}

func TestStatusTracker_RenderFinal(t *testing.T) {
	s := newStatusTracker()
	s.Add("Read", "<code>main.go</code>")
	s.Add("Bash", "<code>go test</code>")

	want := "📄 <code>main.go</code>\n⚡ <code>go test</code>"
	if got := s.RenderFinal(); got != want {
		t.Errorf("RenderFinal() =\n%q\nwant:\n%q", got, want)
	}
}

func TestStatusTracker_AddText(t *testing.T) {
	s := newStatusTracker()
	s.AddText("Let me check that")

	// Spinner only shows on tool entries, not text
	want := "<i>Let me check that</i>"
	if got := s.Render(); got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestStatusTracker_TextColonRenderedAsPeriod(t *testing.T) {
	s := newStatusTracker()
	s.AddText("Let me check:")

	want := "<i>Let me check.</i>"
	if got := s.Render(); got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestStatusTracker_TextAfterTool(t *testing.T) {
	s := newStatusTracker()
	s.Add("Read", "<code>app.go</code>")
	s.AddText("I see the issue")

	want := "📄 <code>app.go</code>\n\n<i>I see the issue</i>"
	if got := s.Render(); got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestStatusTracker_ToolAfterText(t *testing.T) {
	s := newStatusTracker()
	s.AddText("Let me check")
	s.Add("Read", "<code>app.go</code>")

	want := "<i>Let me check</i>\n📄 <code>app.go</code> 🟡"
	if got := s.Render(); got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestStatusTracker_MultipleTexts(t *testing.T) {
	s := newStatusTracker()
	s.AddText("First thought")
	s.AddText("Second thought")

	want := "<i>First thought</i>\n\n<i>Second thought</i>"
	if got := s.Render(); got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestStatusTracker_DropText(t *testing.T) {
	s := newStatusTracker()
	s.Add("Read", "<code>app.go</code>")
	s.AddText("This is the result")

	s.DropText("This is the result")

	want := "📄 <code>app.go</code>"
	if got := s.RenderFinal(); got != want {
		t.Errorf("RenderFinal() = %q, want %q", got, want)
	}
}

func TestStatusTracker_DropText_TrimSpace(t *testing.T) {
	s := newStatusTracker()
	s.AddText("result text")

	s.DropText("  result text\n")

	if got := s.RenderFinal(); got != "" {
		t.Errorf("RenderFinal() after DropText = %q, want empty", got)
	}
}

func TestStatusTracker_DropText_NoMatch(t *testing.T) {
	s := newStatusTracker()
	s.Add("Read", "<code>app.go</code>")
	s.AddText("intermediate text")

	s.DropText("different text")

	want := "📄 <code>app.go</code>\n\n<i>intermediate text</i>"
	if got := s.RenderFinal(); got != want {
		t.Errorf("RenderFinal() = %q, want %q", got, want)
	}
}

func TestStatusTracker_Truncation(t *testing.T) {
	s := newStatusTracker()
	// Each entry is unique (different index) to avoid dedup, ~70 chars each
	for i := 0; i < 200; i++ {
		s.Add("Bash", fmt.Sprintf("<code>command-%03d-with-a-long-argument-to-fill-space</code>", i))
	}

	got := s.Render()
	if len(got) > maxStatusLen+100 {
		t.Errorf("Render() len = %d, should be near or under %d", len(got), maxStatusLen)
	}
	if !strings.Contains(got, "... ") {
		t.Error("truncated render should contain '... ' prefix")
	}
	if !strings.Contains(got, "earlier entries") {
		t.Error("truncated render should mention 'earlier entries'")
	}
}

func TestStatusTracker_NoTruncationWhenShort(t *testing.T) {
	s := newStatusTracker()
	s.Add("Read", "<code>app.go</code>")
	s.Add("Bash", "<code>go test</code>")

	got := s.Render()
	if strings.Contains(got, "...") {
		t.Error("short render should not be truncated")
	}
}
