package file_map

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"plandex-server/syntax"
	"strings"

	"github.com/plandex/plandex/shared"
	"golang.org/x/net/html"
)

func mapSvelte(content []byte) []Definition {
	scriptContent, scriptLang, styleContent := getSvelteScriptAndStyle(content)
	defs := []Definition{}

	if scriptContent != "" {
		var lang shared.TreeSitterLanguage
		if scriptLang == "ts" {
			lang = shared.TreeSitterLanguageTypescript
		} else {
			lang = shared.TreeSitterLanguageJavascript
		}

		parser := syntax.GetParserForLanguage(lang)
		tree, err := parser.ParseCtx(context.Background(), nil, []byte(scriptContent))
		if err != nil {
			log.Printf("mapSvelte - error parsing script content: %v\n", err)
			return defs
		}

		def := Definition{
			Type:      "svelte-script",
			Signature: fmt.Sprintf("<script lang=%q>", scriptLang),
			Children: mapTraditional(Node{
				Lang:   lang,
				TsNode: tree.RootNode(),
				Bytes:  []byte(scriptContent),
			}, nil),
		}

		defs = append(defs, def)
	}

	defs = append(defs, mapMarkup(content)...)

	if styleContent != "" {
		parser := syntax.GetParserForLanguage(shared.TreeSitterLanguageCss)
		tree, err := parser.ParseCtx(context.Background(), nil, []byte(styleContent))
		if err != nil {
			log.Printf("mapSvelte - error parsing style content: %v\n", err)
			return defs
		}

		def := Definition{
			Type:      "svelte-style",
			Signature: "<style>",
			Children: mapTraditional(Node{
				Lang:   shared.TreeSitterLanguageCss,
				TsNode: tree.RootNode(),
				Bytes:  []byte(styleContent),
			}, nil),
		}

		defs = append(defs, def)
	}

	return defs
}

func getSvelteScriptAndStyle(content []byte) (scriptContent, scriptLang, styleContent string) {
	reader := bytes.NewReader(content)
	doc, err := html.Parse(reader)
	if err != nil {
		return "", "", ""
	}

	// Helper function to extract text content from a node
	var getTextContent func(*html.Node) string
	getTextContent = func(n *html.Node) string {
		if n.Type == html.TextNode {
			return n.Data
		}
		var result string
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			result += getTextContent(c)
		}
		return result
	}

	// Helper function to find script and style tags
	var findTags func(*html.Node)
	findTags = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "script" {
				scriptContent = strings.TrimSpace(getTextContent(n))
				// Check for lang attribute
				for _, attr := range n.Attr {
					if attr.Key == "lang" {
						scriptLang = attr.Val
						break
					}
				}
			} else if n.Data == "style" {
				styleContent = strings.TrimSpace(getTextContent(n))
			}
		}

		// Recursively search children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findTags(c)
		}
	}

	findTags(doc)
	return scriptContent, scriptLang, styleContent
}
