package file_map

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

func mapMarkup(content []byte) []Definition {
	reader := bytes.NewReader(content)
	doc, err := html.Parse(reader)
	if err != nil {
		return nil
	}

	var walk func(*html.Node) []Definition
	walk = func(n *html.Node) []Definition {
		var defs []Definition

		if n.Type == html.ElementNode {
			// Only track semantically significant elements
			if isSignificantTag(n.Data) {
				def := Definition{
					Type:      "tag",
					Signature: n.Data,
				}

				// Only include semantic classes/ids
				for _, attr := range n.Attr {
					if attr.Key == "id" {
						def.TagAttrs = append(def.TagAttrs, fmt.Sprintf("#%s", attr.Val))
					} else if attr.Key == "class" {
						classes := strings.Fields(attr.Val)
						if len(classes) > 3 {
							classes = classes[:3]
						}
						def.TagAttrs = append(def.TagAttrs, fmt.Sprintf(".%s", strings.Join(classes, ".")))
					}
				}

				// Get children of this element
				def.Children = []Definition{}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					def.Children = append(def.Children, walk(c)...)
				}

				defs = append(defs, def)
			}
		}

		// Only process siblings for non-significant elements
		if !isSignificantTag(n.Data) {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				defs = append(defs, walk(c)...)
			}
		}

		return defs
	}

	defs := walk(doc)
	defs = consolidateRepeatedTags(defs)
	return defs
}

// Helper function to check if two definitions are equivalent
func areMarkupDefinitionsEqual(a, b Definition) bool {
	if a.Type != b.Type || a.Signature != b.Signature || len(a.TagAttrs) != len(b.TagAttrs) {
		return false
	}

	// Compare attributes
	for i, attr := range a.TagAttrs {
		if attr != b.TagAttrs[i] {
			return false
		}
	}

	return true
}

// Helper function to consolidate repeated tags
func consolidateRepeatedTags(defs []Definition) []Definition {
	var result []Definition

	firstDef := defs[0]
	count := 1
	allEqual := true

	// fmt.Printf("consolidateRepeatedTags: checking %d definitions for equality\n", len(defs))
	// spew.Dump(defs)

	if len(defs) > 1 {
		for i, def := range defs {
			if len(def.Children) > 0 {
				// fmt.Printf("consolidateRepeatedTags: definition %d has children, cannot consolidate\n", i)
				allEqual = false
				break
			}

			if i == 0 {
				continue
			}

			if !areMarkupDefinitionsEqual(firstDef, def) {
				// fmt.Printf("consolidateRepeatedTags: definition %d is not equal to first definition\n", i)
				allEqual = false
				break
			}
			count++
		}
	}

	if allEqual && count > 1 {
		// fmt.Printf("consolidateRepeatedTags: consolidated %d equal definitions\n", count)
		firstDef.TagReps = count
		result = []Definition{firstDef}
	} else {
		// fmt.Printf("consolidateRepeatedTags: definitions not equal, keeping original %d definitions\n", len(defs))
		result = defs
	}

	for i := range result {
		def := &result[i]
		if len(def.Children) > 0 {
			// fmt.Printf("consolidateRepeatedTags: recursively consolidating children of definition %d\n", i)
			def.Children = consolidateRepeatedTags(def.Children)
		}
	}

	return result
}

var significantHtmlTags = map[string]bool{
	"html":     true,
	"head":     true,
	"body":     true,
	"main":     true,
	"nav":      true,
	"header":   true,
	"footer":   true,
	"article":  true,
	"section":  true,
	"form":     true,
	"dialog":   true,
	"template": true,
	"table":    true,
	"div":      true,
	"ul":       true,
	"aside":    true,
}

func isSignificantTag(tag string) bool {
	return significantHtmlTags[tag]
}
