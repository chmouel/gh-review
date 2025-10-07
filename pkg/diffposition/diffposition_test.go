package diffposition

import (
	"testing"
)

func TestMapOldPositionToNew(t *testing.T) {
	tests := []struct {
		name    string
		patch   string
		oldLine int
		want    int
		wantErr bool
	}{
		{
			name: "line before hunk unchanged",
			patch: `@@ -10,3 +10,4 @@
 context
+added
 more context
 end`,
			oldLine: 5,
			want:    5,
			wantErr: false,
		},
		{
			name: "line after hunk with offset",
			patch: `@@ -10,3 +10,4 @@
 context
+added line
 more context
 end`,
			oldLine: 15,
			want:    16, // offset by +1 due to addition
			wantErr: false,
		},
		{
			name: "context line in hunk",
			patch: `@@ -10,3 +10,4 @@
 context
+added
 more context
 end`,
			oldLine: 10,
			want:    10,
			wantErr: false,
		},
		{
			name: "deleted line",
			patch: `@@ -10,3 +10,2 @@
 context
-deleted line
 more context`,
			oldLine: 11,
			want:    -1, // line was deleted
			wantErr: false,
		},
		{
			name: "line after deletion",
			patch: `@@ -10,3 +10,2 @@
 context
-deleted line
 more context`,
			oldLine: 12,
			want:    11, // offset by -1 due to deletion
			wantErr: false,
		},
		{
			name: "multiple hunks",
			patch: `@@ -5,2 +5,3 @@
 line 5
+added at 5
 line 6
@@ -10,2 +11,3 @@
 line 10
+added at 10
 line 11`,
			oldLine: 15,
			want:    17, // offset by +2 (one addition per hunk)
			wantErr: false,
		},
		{
			name: "complex modification",
			patch: `@@ -10,5 +10,4 @@
 context 1
-deleted 1
-deleted 2
+added 1
 context 2
 context 3`,
			oldLine: 15,
			want:    14, // net offset of -1
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapOldPositionToNew(tt.patch, tt.oldLine)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapOldPositionToNew() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MapOldPositionToNew() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapNewPositionToOld(t *testing.T) {
	tests := []struct {
		name    string
		patch   string
		newLine int
		want    int
		wantErr bool
	}{
		{
			name: "line before hunk unchanged",
			patch: `@@ -10,3 +10,4 @@
 context
+added
 more context
 end`,
			newLine: 5,
			want:    5,
			wantErr: false,
		},
		{
			name: "line after hunk with offset",
			patch: `@@ -10,3 +10,4 @@
 context
+added line
 more context
 end`,
			newLine: 16,
			want:    15, // offset by -1 due to addition
			wantErr: false,
		},
		{
			name: "context line in hunk",
			patch: `@@ -10,3 +10,4 @@
 context
+added
 more context
 end`,
			newLine: 10,
			want:    10,
			wantErr: false,
		},
		{
			name: "added line",
			patch: `@@ -10,3 +10,4 @@
 context
+added line
 more context
 end`,
			newLine: 11,
			want:    -1, // line was added, no old line
			wantErr: false,
		},
		{
			name: "line after addition",
			patch: `@@ -10,3 +10,4 @@
 context
+added line
 more context
 end`,
			newLine: 12,
			want:    11, // offset by -1 due to addition
			wantErr: false,
		},
		{
			name: "line after deletion",
			patch: `@@ -10,3 +10,2 @@
 context
-deleted line
 more context`,
			newLine: 11,
			want:    12, // offset by +1 due to deletion
			wantErr: false,
		},
		{
			name: "multiple hunks",
			patch: `@@ -5,2 +5,3 @@
 line 5
+added at 5
 line 6
@@ -10,2 +11,3 @@
 line 10
+added at 10
 line 11`,
			newLine: 17,
			want:    15, // offset by -2 (one addition per hunk)
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapNewPositionToOld(tt.patch, tt.newLine)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapNewPositionToOld() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MapNewPositionToOld() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapPositionRoundTrip(t *testing.T) {
	// Test that mapping old->new->old returns the original (for context lines)
	patch := `@@ -10,5 +10,6 @@
 context 1
 context 2
+added line
 context 3
 context 4
 context 5`

	contextLines := []int{10, 11, 13, 14, 15} // old line numbers for context lines

	for _, oldLine := range contextLines {
		newLine, err := MapOldPositionToNew(patch, oldLine)
		if err != nil {
			t.Errorf("MapOldPositionToNew(%d) error: %v", oldLine, err)
			continue
		}
		if newLine == -1 {
			t.Errorf("MapOldPositionToNew(%d) returned -1, expected valid line", oldLine)
			continue
		}

		backToOld, err := MapNewPositionToOld(patch, newLine)
		if err != nil {
			t.Errorf("MapNewPositionToOld(%d) error: %v", newLine, err)
			continue
		}

		if backToOld != oldLine {
			t.Errorf("Round trip failed: old=%d -> new=%d -> old=%d", oldLine, newLine, backToOld)
		}
	}
}

func TestCalculateCommentPosition(t *testing.T) {
	tests := []struct {
		name         string
		line         int
		originalLine int
		diffHunk     string
		diffSide     DiffSide
		wantOutdated bool
	}{
		{
			name:         "right side added line not outdated",
			line:         11,
			originalLine: 10,
			diffHunk: `@@ -10,2 +10,3 @@
 context
+added line
 more context`,
			diffSide:     DiffSideRight,
			wantOutdated: false,
		},
		{
			name:         "right side context line not outdated",
			line:         10,
			originalLine: 10,
			diffHunk: `@@ -10,3 +10,4 @@
 context
+added
 more context
 end`,
			diffSide:     DiffSideRight,
			wantOutdated: false,
		},
		{
			name:         "left side deleted line outdated",
			line:         11,
			originalLine: 11,
			diffHunk: `@@ -10,3 +10,2 @@
 context
-deleted line
 more context`,
			diffSide:     DiffSideLeft,
			wantOutdated: true,
		},
		{
			name:         "left side context line not outdated",
			line:         10,
			originalLine: 10,
			diffHunk: `@@ -10,3 +10,4 @@
 context
+added
 more context
 end`,
			diffSide:     DiffSideLeft,
			wantOutdated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, err := CalculateCommentPosition(tt.line, tt.originalLine, tt.diffHunk, tt.diffSide)
			if err != nil {
				t.Errorf("CalculateCommentPosition() error = %v", err)
				return
			}

			if pos.IsOutdated != tt.wantOutdated {
				t.Errorf("IsOutdated = %v, want %v", pos.IsOutdated, tt.wantOutdated)
			}

			if pos.Line != tt.line {
				t.Errorf("Line = %v, want %v", pos.Line, tt.line)
			}

			if pos.OriginalLine != tt.originalLine {
				t.Errorf("OriginalLine = %v, want %v", pos.OriginalLine, tt.originalLine)
			}

			if pos.DiffSide != tt.diffSide {
				t.Errorf("DiffSide = %v, want %v", pos.DiffSide, tt.diffSide)
			}
		})
	}
}

func TestGetCommentingRanges(t *testing.T) {
	tests := []struct {
		name    string
		patch   string
		isBase  bool
		want    [][2]int
		wantErr bool
	}{
		{
			name: "base file - deleted and context lines",
			patch: `@@ -10,5 +10,4 @@
 context 1
-deleted 1
-deleted 2
 context 2
 context 3`,
			isBase:  true,
			want:    [][2]int{{10, 14}}, // all 5 old lines (context + deleted)
			wantErr: false,
		},
		{
			name: "modified file - added and context lines",
			patch: `@@ -10,3 +10,5 @@
 context 1
+added 1
+added 2
 context 2
 context 3`,
			isBase:  false,
			want:    [][2]int{{10, 14}}, // all 5 new lines (context + added)
			wantErr: false,
		},
		{
			name: "base file - only context",
			patch: `@@ -10,3 +10,5 @@
 context 1
+added 1
+added 2
 context 2
 context 3`,
			isBase:  true,
			want:    [][2]int{{10, 10}, {11, 12}}, // only context lines, split by additions
			wantErr: false,
		},
		{
			name: "modified file - only context",
			patch: `@@ -10,5 +10,3 @@
 context 1
-deleted 1
-deleted 2
 context 2
 context 3`,
			isBase:  false,
			want:    [][2]int{{10, 10}, {11, 12}}, // only context lines, split by deletions
			wantErr: false,
		},
		{
			name: "multiple hunks",
			patch: `@@ -5,2 +5,3 @@
 line 5
+added
 line 6
@@ -10,2 +11,3 @@
 line 10
+added
 line 11`,
			isBase:  false,
			want:    [][2]int{{5, 7}, {11, 13}}, // two separate ranges
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCommentingRanges(tt.patch, tt.isBase)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommentingRanges() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("GetCommentingRanges() range count = %v, want %v\nGot: %v\nWant: %v",
					len(got), len(tt.want), got, tt.want)
				return
			}

			for i, r := range got {
				if r[0] != tt.want[i][0] || r[1] != tt.want[i][1] {
					t.Errorf("Range[%d] = [%d, %d], want [%d, %d]",
						i, r[0], r[1], tt.want[i][0], tt.want[i][1])
				}
			}
		})
	}
}

func TestDiffSide(t *testing.T) {
	tests := []struct {
		side DiffSide
		want string
	}{
		{DiffSideLeft, "LEFT"},
		{DiffSideRight, "RIGHT"},
	}

	for _, tt := range tests {
		t.Run(string(tt.side), func(t *testing.T) {
			if string(tt.side) != tt.want {
				t.Errorf("DiffSide = %v, want %v", tt.side, tt.want)
			}
		})
	}
}
