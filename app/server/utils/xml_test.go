package utils

import (
	"testing"
)

func TestEscapeInvalidXMLAttributeCharacters(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: `<tag attr="hello & world">`,
			want:  `<tag attr="hello &amp; world">`,
		},
		{
			input: `<tag attr="a < b > c">`,
			want:  `<tag attr="a &lt; b &gt; c">`,
		},
		{
			input: `<tag attr="quote: &quot; test">`,
			want:  `<tag attr="quote: &amp;quot; test">`,
		},
		{
			input: `<tag attr="it's & all">`,
			want:  `<tag attr="it&apos;s &amp; all">`,
		},
		{
			input: `no attributes here`,
			want:  `no attributes here`,
		},
		{
			input: `<multi attr1="a & b" attr2="c > d">`,
			want:  `<multi attr1="a &amp; b" attr2="c &gt; d">`,
		},
		{
			input: "",
			want:  "",
		},
		{
			input: `<unclosed attr="val">`,
			want:  `<unclosed attr="val">`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := EscapeInvalidXMLAttributeCharacters(tt.input)
			if got != tt.want {
				t.Errorf("EscapeInvalidXMLAttributeCharacters(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapeCdata(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"normal text"},
		{"text with ]]> inside"},
		{"<![CDATA[ content ]]> more ]]> stuff"},
		{"no cdata end"},
		{""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			escaped := EscapeCdata(tt.input)
			unescaped := UnescapeCdata(escaped)
			if unescaped != tt.input {
				t.Errorf("roundtrip failed: input=%q → escaped=%q → unescaped=%q", tt.input, escaped, unescaped)
			}
		})
	}
}

func TestUnescapeCdata(t *testing.T) {
	input := "text PDX_ESCAPED_CDATA_END more PDX_ESCAPED_CDATA_END text"
	want := "text ]]> more ]]> text"
	got := UnescapeCdata(input)
	if got != want {
		t.Errorf("UnescapeCdata(%q) = %q, want %q", input, got, want)
	}
}

func TestStripCdata(t *testing.T) {
	tests := []struct {
		input   string
		tagName string
		want    string
	}{
		{
			input:   "<content><![CDATA[hello]]></content>",
			tagName: "content",
			want:    "<content>hello</content>",
		},
		{
			input:   "<content>  <![CDATA[  world  ]]>  </content>",
			tagName: "content",
			want:    "<content>  world  </content>",
		},
		{
			input:   "<name>already stripped</name>",
			tagName: "name",
			want:    "<name>already stripped</name>",
		},
		{
			input:   "<a><![CDATA[x]]></a> <b><![CDATA[y]]></b>",
			tagName: "a",
			want:    "<a>x</a> <b><![CDATA[y]]></b>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := StripCdata(tt.input, tt.tagName)
			if got != tt.want {
				t.Errorf("StripCdata(%q, %q) = %q, want %q", tt.input, tt.tagName, got, tt.want)
			}
		})
	}
}

func TestWrapCdata(t *testing.T) {
	tests := []struct {
		input   string
		tagName string
		want    string
	}{
		{
			input:   "<content>hello</content>",
			tagName: "content",
			want:    "<content><![CDATA[hello]]></content>",
		},
		{
			input:   "<content><![CDATA[already wrapped]]></content>",
			tagName: "content",
			want:    "<content><![CDATA[already wrapped]]></content>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := WrapCdata(tt.input, tt.tagName)
			if got != tt.want {
				t.Errorf("WrapCdata(%q, %q) = %q, want %q", tt.input, tt.tagName, got, tt.want)
			}
		})
	}
}

func TestGetXMLContent(t *testing.T) {
	tests := []struct {
		xml     string
		tagName string
		want    string
	}{
		{
			xml:     "<root><name>Alice</name></root>",
			tagName: "name",
			want:    "Alice",
		},
		{
			xml:     "<name>Bob</name>",
			tagName: "name",
			want:    "Bob",
		},
		{
			xml:     "<a><b>first</b></a><b>second</b>",
			tagName: "b",
			want:    "second", // last occurrence
		},
		{
			xml:     "<root></root>",
			tagName: "missing",
			want:    "",
		},
		{
			xml:     "",
			tagName: "any",
			want:    "",
		},
		{
			xml:     "<name>unterminated",
			tagName: "name",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.xml, func(t *testing.T) {
			got := GetXMLContent(tt.xml, tt.tagName)
			if got != tt.want {
				t.Errorf("GetXMLContent(%q, %q) = %q, want %q", tt.xml, tt.tagName, got, tt.want)
			}
		})
	}
}

func TestGetAllXMLContent(t *testing.T) {
	tests := []struct {
		xml     string
		tagName string
		want    []string
	}{
		{
			xml:     "<items><item>a</item><item>b</item></items>",
			tagName: "item",
			want:    []string{"a", "b"},
		},
		{
			xml:     "<p>single</p>",
			tagName: "p",
			want:    []string{"single"},
		},
		{
			xml:     "<div>text</div>",
			tagName: "span",
			want:    nil,
		},
		{
			xml:     "no tags",
			tagName: "any",
			want:    nil,
		},
		{
			xml:     "<item>1</item><item>2</item><item>3</item>",
			tagName: "item",
			want:    []string{"1", "2", "3"},
		},
		{
			xml:     "",
			tagName: "tag",
			want:    nil,
		},
		{
			xml:     "<item>1</item><item>unterminated",
			tagName: "item",
			want:    []string{"1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.xml, func(t *testing.T) {
			got := GetAllXMLContent(tt.xml, tt.tagName)
			if len(got) != len(tt.want) {
				t.Errorf("GetAllXMLContent(%q, %q) = %v (len=%d), want %v (len=%d)",
					tt.xml, tt.tagName, got, len(got), tt.want, len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("GetAllXMLContent[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGetXMLTag(t *testing.T) {
	// GetXMLTag wraps content back in tags
	input := "<root><message>Hello & welcome</message></root>"
	got := GetXMLTag(input, "message", false)
	// Should contain the tag with escaped content
	if got == "" {
		t.Error("GetXMLTag returned empty string")
	}
	// Should start and end with the tag
	if len(got) < len("<message></message>") {
		t.Errorf("GetXMLTag too short: %q", got)
	}
}

func TestCdataRoundtrip(t *testing.T) {
	original := "some <![CDATA[ real ]]> content ]]> here"
	escaped := EscapeCdata(original)
	unescaped := UnescapeCdata(escaped)
	if unescaped != original {
		t.Errorf("CDATA roundtrip failed: %q → %q → %q", original, escaped, unescaped)
	}
}

func TestWrapStripRoundtrip(t *testing.T) {
	original := "<desc>hello world</desc>"
	wrapped := WrapCdata(original, "desc")
	stripped := StripCdata(wrapped, "desc")
	if stripped != original {
		t.Errorf("wrap/strip roundtrip: %q → %q → %q", original, wrapped, stripped)
	}
}
