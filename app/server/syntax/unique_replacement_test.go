package syntax

import (
	"testing"
)

func TestFindUniqueReplacement(t *testing.T) {
	tests := []struct {
		name         string
		originalFile string
		old          string
		want         string
	}{
		{
			name:         "perfect single match",
			originalFile: "prefix ABC123DEF suffix",
			old:          "ABC123DEF",
			want:         "ABC123DEF",
		},
		{
			name:         "match with error in middle",
			originalFile: "prefix ABC999DEF suffix",
			old:          "ABC123DEF",
			want:         "ABC999DEF",
		},
		{
			name:         "multiple instances but unique boundaries",
			originalFile: "ABC123XYZ ABC456XYZ ABC789DEF",
			old:          "ABC789DEF",
			want:         "ABC789DEF",
		},
		{
			name:         "no match at all",
			originalFile: "completely different text",
			old:          "ABC123DEF",
			want:         "",
		},
		{
			name:         "multiple complete matches",
			originalFile: "ABC123DEF ABC123DEF",
			old:          "ABC123DEF",
			want:         "", // should fail because not unique
		},
		{
			name:         "ambiguous boundaries",
			originalFile: "ABC123DEF ABC456DEF",
			old:          "ABC789DEF",
			want:         "", // should fail because multiple possible matches
		},
		{
			name:         "match with very different middle",
			originalFile: "prefix ABCCOMPLETELY_DIFFERENT_TEXTDEF suffix",
			old:          "ABC123DEF",
			want:         "ABCCOMPLETELY_DIFFERENT_TEXTDEF",
		},
		{
			name:         "unique match near identical text",
			originalFile: "ABCDEF ABC123DEF ABCXEF",
			old:          "ABC123DEF",
			want:         "ABC123DEF",
		},
		{
			name:         "identical start/end patterns",
			originalFile: "AAA123AAA AAA456AAA",
			old:          "AAA789AAA",
			want:         "", // should fail because boundaries are ambiguous
		},
		{
			name:         "overlapping patterns",
			originalFile: "ABCABCDEF",
			old:          "ABCDEF",
			want:         "ABCDEF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindUniqueReplacement(tt.originalFile, tt.old)
			if got != tt.want {
				t.Errorf("FindUniqueReplacement(%q, %q) = %q, want %q",
					tt.originalFile, tt.old, got, tt.want)
			}
		})
	}
}
