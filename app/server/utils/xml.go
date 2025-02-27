package utils

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

func StripCdata(xmlString, tagName string) string {
	openTag := "<" + tagName + ">"
	closeTag := "</" + tagName + ">"
	xmlString = regexp.MustCompile(openTag+`\s*<!\[CDATA\[`).ReplaceAllString(xmlString, openTag)
	xmlString = regexp.MustCompile(`]]>\s*`+closeTag).ReplaceAllString(xmlString, closeTag)
	return xmlString
}

func WrapCdata(xmlString, tagName string) string {
	openTag := "<" + tagName + ">"
	closeTag := "</" + tagName + ">"
	xmlString = StripCdata(xmlString, tagName)

	xmlString = strings.ReplaceAll(xmlString, openTag, openTag+"<![CDATA[")
	xmlString = strings.ReplaceAll(xmlString, closeTag, "]]>"+closeTag)

	return xmlString
}

func GetXMLTag(xmlString, tagName string, wrapCdata bool) string {
	openTag := "<" + tagName + ">"
	closeTag := "</" + tagName + ">"

	// Get everything after the last opening tag
	split := strings.Split(xmlString, openTag)
	if len(split) < 2 {
		return ""
	}
	afterOpenTag := split[len(split)-1]

	// Get everything before the first closing tag
	split2 := strings.Split(afterOpenTag, closeTag)
	if len(split2) < 1 {
		return ""
	}

	processedXml := openTag + EscapeInvalidXMLAttributeCharacters(split2[0]) + closeTag

	if wrapCdata {
		processedXml = WrapCdata(processedXml, tagName)
	}

	return processedXml
}

func GetXMLContent(xmlString, tagName string) string {
	openTag := "<" + tagName + ">"
	closeTag := "</" + tagName + ">"

	// Get everything after the last opening tag
	split := strings.Split(xmlString, openTag)
	if len(split) < 2 {
		return ""
	}
	afterOpenTag := split[len(split)-1]

	// Get everything before the first closing tag
	split2 := strings.Split(afterOpenTag, closeTag)
	if len(split2) < 1 {
		return ""
	}

	return split2[0]
}

// GetAllXMLContent returns all occurrences of content between the specified XML tags
// as an array of strings. Returns an empty array if no matches are found.
func GetAllXMLContent(xmlString, tagName string) []string {
	var results []string
	openTag := "<" + tagName + ">"
	closeTag := "</" + tagName + ">"

	// Split by opening tag
	parts := strings.Split(xmlString, openTag)

	// Skip the first part (it's before any opening tag)
	for i := 1; i < len(parts); i++ {
		// Split by closing tag
		subParts := strings.Split(parts[i], closeTag)
		if len(subParts) > 0 {
			// The content is before the first closing tag
			results = append(results, subParts[0])
		}
	}

	return results
}
