package plan

import (
	"regexp"
	"strings"
)

func EscapeInvalidXMLAttributeCharacters(xmlString string) string {
	// Regular expression to match content inside double quotes, but not the quotes themselves
	re := regexp.MustCompile(`"([^"]*?)"`)
	return re.ReplaceAllStringFunc(xmlString, func(attrValue string) string {
		// Extract the content inside the quotes (removing the enclosing quotes)
		content := attrValue[1 : len(attrValue)-1]

		// Escape the content inside the quotes
		escaped := strings.ReplaceAll(content, "&", "&amp;")
		escaped = strings.ReplaceAll(escaped, "<", "&lt;")
		escaped = strings.ReplaceAll(escaped, ">", "&gt;")
		escaped = strings.ReplaceAll(escaped, `"`, "&quot;")
		escaped = strings.ReplaceAll(escaped, "'", "&apos;")

		// Re-wrap the escaped content in quotes
		return `"` + escaped + `"`
	})
}

func EscapeCdata(xmlString string) string {
	escaped := strings.ReplaceAll(xmlString, "]]>", "PDX_ESCAPED_CDATA_END")
	return escaped
}

func UnescapeCdata(xmlString string) string {
	escaped := strings.ReplaceAll(xmlString, "PDX_ESCAPED_CDATA_END", "]]>")
	return escaped
}
