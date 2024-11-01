package types

import "encoding/xml"

type SemanticAnchors struct {
	XMLName xml.Name `xml:"SemanticAnchors"`
	Anchors []Anchor `xml:"Anchor"`
}

type Anchor struct {
	Reasoning    string `xml:"reasoning,attr"`
	ProposedLine string `xml:"proposedLine,attr"`
	OriginalLine string `xml:"originalLine,attr"`
}
