package parse

import (
	"testing"
)

func TestParseMoveFiles(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []FileMove
	}{
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name: "single move",
			input: `### Move Files
- ` + "`source/file.ts` → `dest/file.ts`",
			expected: []FileMove{
				{Source: "source/file.ts", Destination: "dest/file.ts"},
			},
		},
		{
			name: "multiple moves",
			input: `### Move Files
- ` + "`src/components/button.tsx` → `components/button.tsx`" + `
- ` + "`old/utils.ts` → `utils/index.ts`",
			expected: []FileMove{
				{Source: "src/components/button.tsx", Destination: "components/button.tsx"},
				{Source: "old/utils.ts", Destination: "utils/index.ts"},
			},
		},
		{
			name: "ignores invalid lines",
			input: `### Move Files
- ` + "`valid/source.ts` → `valid/dest.ts`" + `
- not properly formatted
- ` + "`missing/arrow`" + `
Some other content`,
			expected: []FileMove{
				{Source: "valid/source.ts", Destination: "valid/dest.ts"},
			},
		},
		{
			name: "handles empty lines",
			input: `### Move Files
- ` + "`src/file1.ts` → `dest/file1.ts`" + `

- ` + "`src/file2.ts` → `dest/file2.ts`",
			expected: []FileMove{
				{Source: "src/file1.ts", Destination: "dest/file1.ts"},
				{Source: "src/file2.ts", Destination: "dest/file2.ts"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseMoveFiles(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("ParseMoveFiles() returned %d moves, want %d", len(got), len(tt.expected))
				return
			}
			for i := range got {
				if got[i].Source != tt.expected[i].Source {
					t.Errorf("Move[%d].Source = %q, want %q", i, got[i].Source, tt.expected[i].Source)
				}
				if got[i].Destination != tt.expected[i].Destination {
					t.Errorf("Move[%d].Destination = %q, want %q", i, got[i].Destination, tt.expected[i].Destination)
				}
			}
		})
	}
}

func TestParseRemoveFiles(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name: "single file",
			input: `### Remove Files
- ` + "`components/button.tsx`",
			expected: []string{"components/button.tsx"},
		},
		{
			name: "multiple files",
			input: `### Remove Files
- ` + "`src/file1.ts`" + `
- ` + "`src/file2.ts`",
			expected: []string{"src/file1.ts", "src/file2.ts"},
		},
		{
			name: "ignores invalid lines",
			input: `### Remove Files
- ` + "`valid/file.ts`" + `
- missing backticks
Some other content`,
			expected: []string{"valid/file.ts"},
		},
		{
			name: "handles empty lines",
			input: `### Remove Files
- ` + "`file1.ts`" + `

- ` + "`file2.ts`",
			expected: []string{"file1.ts", "file2.ts"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseRemoveFiles(tt.input)
			if !sliceEqual(got, tt.expected) {
				t.Errorf("ParseRemoveFiles() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseResetChanges(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name: "single file",
			input: `### Reset Changes
- ` + "`components/button.tsx`",
			expected: []string{"components/button.tsx"},
		},
		{
			name: "multiple files",
			input: `### Reset Changes
- ` + "`src/file1.ts`" + `
- ` + "`src/file2.ts`",
			expected: []string{"src/file1.ts", "src/file2.ts"},
		},
		{
			name: "ignores invalid lines",
			input: `### Reset Changes
- ` + "`valid/file.ts`" + `
- missing backticks
Some other content`,
			expected: []string{"valid/file.ts"},
		},
		{
			name: "handles empty lines",
			input: `### Reset Changes
- ` + "`file1.ts`" + `

- ` + "`file2.ts`",
			expected: []string{"file1.ts", "file2.ts"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseResetChanges(tt.input)
			if !sliceEqual(got, tt.expected) {
				t.Errorf("ParseResetChanges() = %v, want %v", got, tt.expected)
			}
		})
	}
}
