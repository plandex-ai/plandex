package plan

import (
	"fmt"
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

func GetXMLTag(xmlString, tagName string) (string, error) {
	openTag := "<" + tagName + ">"
	closeTag := "</" + tagName + ">"

	split := strings.Split(xmlString, openTag)
	if len(split) != 2 {
		return "", fmt.Errorf("error processing xml")
	}
	afterOpenTag := split[1]

	split2 := strings.Split(afterOpenTag, closeTag)
	if len(split2) != 2 {
		return "", fmt.Errorf("error processing xml")
	}

	processedXml := openTag + EscapeInvalidXMLAttributeCharacters(split2[0]) + closeTag

	return processedXml, nil
}
