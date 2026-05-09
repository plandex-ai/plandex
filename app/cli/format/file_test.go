package format

import "testing"

func TestGetFileNameWithoutExt(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Basic extension removal and normalization
		{"MyFile.go", "myfile"},
		{"hello_world.rs", "hello-world"},
		{"Foo Bar.md", "foo-bar"},
		{"some/path/file.txt", "some-path-file"},

		// Special characters
		{"file.name.with.dots.js", "file-name-with-dots"},
		{`windows\path\file.go`, "windows-path-file"},
		{"it's a file.txt", "its-a-file"},
		{"back`tick.md", "backtick"},
		{`"quoted".txt`, "quoted"},

		// Multiple underscores
		{"__init__.py", "--init--"},
		{"test___file.go", "test---file"},

		// No extension
		{"Makefile", "makefile"},
		{"README", "readme"},

		// Edge cases
		{"", ""},
		// ".hidden" has no non-extension part, so returns ""
		{".hidden", ""},
		{".hidden.txt", "-hidden"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := GetFileNameWithoutExt(tt.input)
			if got != tt.want {
				t.Errorf("GetFileNameWithoutExt(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetFileNameWithoutExtIdempotent(t *testing.T) {
	// Running the function twice should give the same result
	inputs := []string{
		"Hello World.go",
		"foo_bar_baz.rs",
		"path/to/file.txt",
	}

	for _, input := range inputs {
		first := GetFileNameWithoutExt(input)
		second := GetFileNameWithoutExt(first)
		if first != second {
			t.Errorf("not idempotent: GetFileNameWithoutExt(%q) = %q, but GetFileNameWithoutExt(%q) = %q",
				input, first, first, second)
		}
	}
}
