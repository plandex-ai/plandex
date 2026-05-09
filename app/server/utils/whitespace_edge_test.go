package utils

import (
	"testing"
)

func TestStripAddedBlankLines_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		orig string
		upd  string
		want string
	}{
		{
			name: "both empty",
			orig: "",
			upd:  "",
			want: "",
		},
		{
			name: "orig empty, upd has content",
			orig: "",
			upd:  "hello\nworld\n",
			want: "hello\nworld\n",
		},
		{
			name: "only whitespace lines in orig",
			orig: "\n\n\n",
			upd:  "\n\nsome content\n\n\n",
			want: "\n\nsome content\n\n\n",
		},
		{
			name: "spaces considered blank, surplus leading/trailing stripped",
			orig: "a\nb\n",
			upd:  "   \n   \na\nb\n   \n",
			want: "a\nb\n   ",
		},
		{
			name: "tabs considered blank, surplus trailing stripped",
			orig: "content\n",
			upd:  "\t\ncontent\n\t\n",
			want: "content\n\t",
		},
		{
			name: "upd has fewer blank lines",
			orig: "\n\ncontent\n\n",
			upd:  "\ncontent\n",
			want: "\ncontent\n",
		},
		{
			name: "no blank lines anywhere",
			orig: "a\nb\nc\n",
			upd:  "a\nb\nc\n",
			want: "a\nb\nc\n",
		},
		{
			name: "surplus blank lines removed, content preserved",
			orig: "hello\n\nworld\n",
			upd:  "\n\n\nhello\n\nworld\n\n\n",
			want: "hello\n\nworld\n",
		},
		{
			name: "single line",
			orig: "line\n",
			upd:  "\n\nline\n\n",
			want: "line\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripAddedBlankLines(tt.orig, tt.upd)
			if got != tt.want {
				t.Errorf("StripAddedBlankLines(%q, %q) = %q, want %q",
					tt.orig, tt.upd, got, tt.want)
			}
		})
	}
}
