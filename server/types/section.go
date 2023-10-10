package types

// Section represents a section in a text document
// It can have sub-sections as well
type Section struct {
	Name       string
	Content    string
	Subsections []Section
}

// SectionizeResponse represents the response from the sectionize function
// It consists of sections for the document
type SectionizeResponse struct {
	Sections []Section
}
