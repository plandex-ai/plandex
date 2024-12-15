package parse

import (
	"plandex-server/db"
	"testing"
)

func TestParseSubtasks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []*db.Subtask
	}{
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name: "single task without description",
			input: `### Tasks
1. Create a new file`,
			expected: []*db.Subtask{
				{
					Title:       "Create a new file",
					Description: "",
					UsesFiles:   nil,
				},
			},
		},
		{
			name: "multiple tasks with descriptions and uses",
			input: `### Tasks
1. Create config file
- Will store application settings
- Contains environment variables
Uses: ` + "`config/settings.yml`" + `, ` + "`config/defaults.yml`" + `

2. Update main function
- Add configuration loading
Uses: ` + "`main.go`",
			expected: []*db.Subtask{
				{
					Title:       "Create config file",
					Description: "Will store application settings\nContains environment variables",
					UsesFiles:   []string{"config/settings.yml", "config/defaults.yml"},
				},
				{
					Title:       "Update main function",
					Description: "Add configuration loading",
					UsesFiles:   []string{"main.go"},
				},
			},
		},
		{
			name: "alternative task header",
			input: `### Task
1. Simple task`,
			expected: []*db.Subtask{
				{
					Title:       "Simple task",
					Description: "",
					UsesFiles:   nil,
				},
			},
		},
		{
			name: "tasks with empty lines between",
			input: `### Tasks
1. First task
- Description one

2. Second task
- Description two`,
			expected: []*db.Subtask{
				{
					Title:       "First task",
					Description: "Description one",
					UsesFiles:   nil,
				},
				{
					Title:       "Second task",
					Description: "Description two",
					UsesFiles:   nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSubtasks(tt.input)

			if len(got) != len(tt.expected) {
				t.Errorf("ParseSubtasks() returned %d subtasks, want %d", len(got), len(tt.expected))
				return
			}

			for i := range got {
				if got[i].Title != tt.expected[i].Title {
					t.Errorf("Subtask[%d].Title = %q, want %q", i, got[i].Title, tt.expected[i].Title)
				}
				if got[i].Description != tt.expected[i].Description {
					t.Errorf("Subtask[%d].Description = %q, want %q", i, got[i].Description, tt.expected[i].Description)
				}
				if !sliceEqual(got[i].UsesFiles, tt.expected[i].UsesFiles) {
					t.Errorf("Subtask[%d].UsesFiles = %v, want %v", i, got[i].UsesFiles, tt.expected[i].UsesFiles)
				}
			}
		})
	}
}

// sliceEqual compares two string slices for equality
func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
