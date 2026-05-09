package url

import "testing"

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"https://example.com", true},
		{"http://example.com/path?q=1", true},
		{"https://sub.domain.example.com/path/to/resource", true},
		{"ftp://files.example.com", true},
		{"https://example.com:8080/path", true},
		{"https://user:pass@example.com", true},
		{"", false},
		{"not-a-url", false},
		{"example.com", false},        // no scheme
		{"/relative/path", false},       // no scheme, no host
		{"http://", false},             // no host
		{"://missing-scheme.com", false},
		{"   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsValidURL(tt.input)
			if got != tt.want {
				t.Errorf("IsValidURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://example.com/path", "example.com_path"},
		{"http://example.com/path?q=1&x=2", "example.com_path_q_1_x_2"},
		{"https://example.com:8080/path#anchor", "example.com_8080_path_anchor"},
		{"https://example.com/path%20with%20spaces", "example.com_path_20with_20spaces"},
		{"https://example.com/file*.txt", "example.com_file_.txt"},
		{"https://example.com/a/b/c", "example.com_a_b_c"},
		{"just-a-string", "just-a-string"}, // no protocol to strip
		{"ftp://files.example.com", "files.example.com"},
		{"https://example.com/path with spaces", "example.com_path_with_spaces"},
		{"https://example.com/path=value&key=val", "example.com_path_value_key_val"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeURL(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractTextualContent(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "simple paragraph",
			html: "<html><body><p>Hello World</p></body></html>",
			want: "Hello World",
		},
		{
			name: "multiple elements",
			html: "<html><body><h1>Title</h1><p>Content</p></body></html>",
			want: "TitleContent",
		},
		{
			name: "empty body",
			html: "<html><body></body></html>",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTextualContent(tt.html)
			if got != tt.want {
				t.Errorf("ExtractTextualContent(%q) = %q, want %q", tt.html, got, tt.want)
			}
		})
	}
}
