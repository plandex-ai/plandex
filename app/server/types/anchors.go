package types

import "encoding/xml"

type SemanticAnchorsTag struct {
	XMLName xml.Name    `xml:"PlandexSemanticAnchors"`
	Anchors []AnchorTag `xml:"Anchor"`
}

type AnchorTag struct {
	Reasoning    string `xml:"reasoning,attr"`
	ProposedLine string `xml:"proposedLine,attr"`
	OriginalLine string `xml:"originalLine,attr"`
}

type SummaryTag struct {
	XMLName xml.Name `xml:"PlandexSummary"`
	Content string   `xml:",chardata"`
}
