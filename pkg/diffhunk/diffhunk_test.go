package diffhunk

import (
	"strings"
	"testing"
)

func TestParseDiffHunk(t *testing.T) {
	tests := []struct {
		name    string
		hunk    string
		want    *DiffHunk
		wantErr bool
	}{
		{
			name: "simple addition",
			hunk: `@@ -1,3 +1,4 @@
 context line 1
 context line 2
+added line
 context line 3`,
			want: &DiffHunk{
				OldStart: 1,
				OldLines: 3,
				NewStart: 1,
				NewLines: 4,
				Lines: []*DiffLine{
					{Type: Context, OldLineNumber: 1, NewLineNumber: 1, Text: "context line 1", PositionInHunk: 0},
					{Type: Context, OldLineNumber: 2, NewLineNumber: 2, Text: "context line 2", PositionInHunk: 1},
					{Type: Add, OldLineNumber: -1, NewLineNumber: 3, Text: "added line", PositionInHunk: 2},
					{Type: Context, OldLineNumber: 3, NewLineNumber: 4, Text: "context line 3", PositionInHunk: 3},
				},
			},
			wantErr: false,
		},
		{
			name: "simple deletion",
			hunk: `@@ -1,4 +1,3 @@
 context line 1
 context line 2
-deleted line
 context line 3`,
			want: &DiffHunk{
				OldStart: 1,
				OldLines: 4,
				NewStart: 1,
				NewLines: 3,
				Lines: []*DiffLine{
					{Type: Context, OldLineNumber: 1, NewLineNumber: 1, Text: "context line 1", PositionInHunk: 0},
					{Type: Context, OldLineNumber: 2, NewLineNumber: 2, Text: "context line 2", PositionInHunk: 1},
					{Type: Delete, OldLineNumber: 3, NewLineNumber: -1, Text: "deleted line", PositionInHunk: 2},
					{Type: Context, OldLineNumber: 4, NewLineNumber: 3, Text: "context line 3", PositionInHunk: 3},
				},
			},
			wantErr: false,
		},
		{
			name: "modification (delete + add)",
			hunk: `@@ -10,3 +10,3 @@
 context before
-old line
+new line
 context after`,
			want: &DiffHunk{
				OldStart: 10,
				OldLines: 3,
				NewStart: 10,
				NewLines: 3,
				Lines: []*DiffLine{
					{Type: Context, OldLineNumber: 10, NewLineNumber: 10, Text: "context before", PositionInHunk: 0},
					{Type: Delete, OldLineNumber: 11, NewLineNumber: -1, Text: "old line", PositionInHunk: 1},
					{Type: Add, OldLineNumber: -1, NewLineNumber: 11, Text: "new line", PositionInHunk: 2},
					{Type: Context, OldLineNumber: 12, NewLineNumber: 12, Text: "context after", PositionInHunk: 3},
				},
			},
			wantErr: false,
		},
		{
			name: "single line count (implicit 1)",
			hunk: `@@ -5 +5 @@
-old
+new`,
			want: &DiffHunk{
				OldStart: 5,
				OldLines: 1,
				NewStart: 5,
				NewLines: 1,
				Lines: []*DiffLine{
					{Type: Delete, OldLineNumber: 5, NewLineNumber: -1, Text: "old", PositionInHunk: 0},
					{Type: Add, OldLineNumber: -1, NewLineNumber: 5, Text: "new", PositionInHunk: 1},
				},
			},
			wantErr: false,
		},
		{
			name: "no newline marker",
			hunk: `@@ -1,2 +1,2 @@
 line 1
-line 2
\ No newline at end of file
+line 2 fixed
\ No newline at end of file`,
			want: &DiffHunk{
				OldStart: 1,
				OldLines: 2,
				NewStart: 1,
				NewLines: 2,
				Lines: []*DiffLine{
					{Type: Context, OldLineNumber: 1, NewLineNumber: 1, Text: "line 1", PositionInHunk: 0},
					{Type: Delete, OldLineNumber: 2, NewLineNumber: -1, Text: "line 2", PositionInHunk: 1},
					{Type: Control, OldLineNumber: -1, NewLineNumber: -1, Text: `\ No newline at end of file`, PositionInHunk: 2},
					{Type: Add, OldLineNumber: -1, NewLineNumber: 2, Text: "line 2 fixed", PositionInHunk: 3},
					{Type: Control, OldLineNumber: -1, NewLineNumber: -1, Text: `\ No newline at end of file`, PositionInHunk: 4},
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid header",
			hunk:    `not a valid hunk`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty hunk",
			hunk:    ``,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDiffHunk(tt.hunk)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDiffHunk() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Compare hunk metadata
			if got.OldStart != tt.want.OldStart {
				t.Errorf("OldStart = %v, want %v", got.OldStart, tt.want.OldStart)
			}
			if got.OldLines != tt.want.OldLines {
				t.Errorf("OldLines = %v, want %v", got.OldLines, tt.want.OldLines)
			}
			if got.NewStart != tt.want.NewStart {
				t.Errorf("NewStart = %v, want %v", got.NewStart, tt.want.NewStart)
			}
			if got.NewLines != tt.want.NewLines {
				t.Errorf("NewLines = %v, want %v", got.NewLines, tt.want.NewLines)
			}

			// Compare lines
			if len(got.Lines) != len(tt.want.Lines) {
				t.Errorf("Lines count = %v, want %v", len(got.Lines), len(tt.want.Lines))
				return
			}

			for i, line := range got.Lines {
				wantLine := tt.want.Lines[i]
				if line.Type != wantLine.Type {
					t.Errorf("Line[%d].Type = %v, want %v", i, line.Type, wantLine.Type)
				}
				if line.OldLineNumber != wantLine.OldLineNumber {
					t.Errorf("Line[%d].OldLineNumber = %v, want %v", i, line.OldLineNumber, wantLine.OldLineNumber)
				}
				if line.NewLineNumber != wantLine.NewLineNumber {
					t.Errorf("Line[%d].NewLineNumber = %v, want %v", i, line.NewLineNumber, wantLine.NewLineNumber)
				}
				if line.Text != wantLine.Text {
					t.Errorf("Line[%d].Text = %q, want %q", i, line.Text, wantLine.Text)
				}
				if line.PositionInHunk != wantLine.PositionInHunk {
					t.Errorf("Line[%d].PositionInHunk = %v, want %v", i, line.PositionInHunk, wantLine.PositionInHunk)
				}
			}
		})
	}
}

func TestParsePatch(t *testing.T) {
	tests := []struct {
		name      string
		patch     string
		wantCount int
		wantErr   bool
	}{
		{
			name: "single hunk",
			patch: `@@ -1,3 +1,4 @@
 line 1
+added
 line 2
 line 3`,
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "multiple hunks",
			patch: `@@ -1,3 +1,4 @@
 line 1
+added 1
 line 2
 line 3
@@ -10,2 +11,3 @@
 line 10
+added 2
 line 11`,
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "full patch with headers",
			patch: `diff --git a/file.go b/file.go
index 1234567..abcdefg 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 package main
+import "fmt"

 func main() {`,
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "no hunks",
			patch:     `some random text without hunks`,
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePatch(tt.patch)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(got) != tt.wantCount {
				t.Errorf("ParsePatch() hunk count = %v, want %v", len(got), tt.wantCount)
			}
		})
	}
}

func TestGetDiffLineByPosition(t *testing.T) {
	patch := `@@ -1,5 +1,6 @@
 line 1
 line 2
+added line
 line 3
 line 4
 line 5`

	hunks, err := ParsePatch(patch)
	if err != nil {
		t.Fatalf("Failed to parse patch: %v", err)
	}

	tests := []struct {
		name     string
		position int
		wantText string
		wantType DiffChangeType
	}{
		{name: "first line", position: 1, wantText: "line 1", wantType: Context},
		{name: "second line", position: 2, wantText: "line 2", wantType: Context},
		{name: "added line", position: 3, wantText: "added line", wantType: Add},
		{name: "after addition", position: 4, wantText: "line 3", wantType: Context},
		{name: "last line", position: 6, wantText: "line 5", wantType: Context},
		{name: "out of range", position: 100, wantText: "", wantType: Context},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDiffLineByPosition(hunks, tt.position)
			if tt.wantText == "" {
				if got != nil {
					t.Errorf("GetDiffLineByPosition() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("GetDiffLineByPosition() = nil, want line")
			}
			if got.Text != tt.wantText {
				t.Errorf("Text = %q, want %q", got.Text, tt.wantText)
			}
			if got.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", got.Type, tt.wantType)
			}
		})
	}
}

func TestGetAddedLines(t *testing.T) {
	tests := []struct {
		name string
		hunk string
		want []string
	}{
		{
			name: "single addition",
			hunk: `@@ -1,2 +1,3 @@
 line 1
+added line
 line 2`,
			want: []string{"added line"},
		},
		{
			name: "multiple additions",
			hunk: `@@ -1,2 +1,4 @@
 line 1
+added 1
+added 2
 line 2`,
			want: []string{"added 1", "added 2"},
		},
		{
			name: "no additions",
			hunk: `@@ -1,2 +1,2 @@
 line 1
 line 2`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetAddedLines(tt.hunk)
			if len(got) != len(tt.want) {
				t.Errorf("GetAddedLines() count = %v, want %v", len(got), len(tt.want))
				return
			}
			for i, line := range got {
				if line != tt.want[i] {
					t.Errorf("Line[%d] = %q, want %q", i, line, tt.want[i])
				}
			}
		})
	}
}

func TestGetRemovedLines(t *testing.T) {
	tests := []struct {
		name string
		hunk string
		want []string
	}{
		{
			name: "single deletion",
			hunk: `@@ -1,3 +1,2 @@
 line 1
-deleted line
 line 2`,
			want: []string{"deleted line"},
		},
		{
			name: "multiple deletions",
			hunk: `@@ -1,4 +1,2 @@
 line 1
-deleted 1
-deleted 2
 line 2`,
			want: []string{"deleted 1", "deleted 2"},
		},
		{
			name: "no deletions",
			hunk: `@@ -1,2 +1,2 @@
 line 1
 line 2`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRemovedLines(tt.hunk)
			if len(got) != len(tt.want) {
				t.Errorf("GetRemovedLines() count = %v, want %v", len(got), len(tt.want))
				return
			}
			for i, line := range got {
				if line != tt.want[i] {
					t.Errorf("Line[%d] = %q, want %q", i, line, tt.want[i])
				}
			}
		})
	}
}

func TestGetModifiedContentFromDiffHunk(t *testing.T) {
	tests := []struct {
		name     string
		original string
		patch    string
		want     string
		wantErr  bool
	}{
		{
			name:     "simple addition",
			original: "line 1\nline 2\nline 3",
			patch: `@@ -1,3 +1,4 @@
 line 1
+added line
 line 2
 line 3`,
			want:    "line 1\nadded line\nline 2\nline 3",
			wantErr: false,
		},
		{
			name:     "simple deletion",
			original: "line 1\nline 2\nline 3",
			patch: `@@ -1,3 +1,2 @@
 line 1
-line 2
 line 3`,
			want:    "line 1\nline 3",
			wantErr: false,
		},
		{
			name:     "modification",
			original: "line 1\nold line\nline 3",
			patch: `@@ -1,3 +1,3 @@
 line 1
-old line
+new line
 line 3`,
			want:    "line 1\nnew line\nline 3",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetModifiedContentFromDiffHunk(tt.original, tt.patch)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetModifiedContentFromDiffHunk() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Normalize line endings for comparison
			gotLines := strings.Split(strings.TrimRight(got, "\n"), "\n")
			wantLines := strings.Split(strings.TrimRight(tt.want, "\n"), "\n")

			if len(gotLines) != len(wantLines) {
				t.Errorf("Line count = %v, want %v\nGot:\n%s\nWant:\n%s",
					len(gotLines), len(wantLines), got, tt.want)
				return
			}

			for i := range gotLines {
				if gotLines[i] != wantLines[i] {
					t.Errorf("Line[%d] = %q, want %q", i, gotLines[i], wantLines[i])
				}
			}
		})
	}
}

func TestGetZeroBased(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{name: "zero stays zero", input: 0, want: 0},
		{name: "one becomes zero", input: 1, want: 0},
		{name: "ten becomes nine", input: 10, want: 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetZeroBased(tt.input); got != tt.want {
				t.Errorf("GetZeroBased() = %v, want %v", got, tt.want)
			}
		})
	}
}
