package types

import "encoding/xml"

type SemanticAnchorsTag struct {
	XMLName xml.Name    `xml:"PlandexSemanticAnchors"`
	Anchors []AnchorTag `xml:"Anchor"`
}

type AnchorTag struct {
	// Reasoning    string `xml:"reasoning,attr"` // better to leave this out since it can cause problems with unmarshalling and isn't used after parsing
	ProposedLine string `xml:"proposedLine,attr"`
	OriginalLine string `xml:"originalLine,attr"`
}

type SummaryTag struct {
	XMLName xml.Name `xml:"PlandexSummary"`
	Content string   `xml:",chardata"`
}
