package cpe

// List has cpe-item list
// https://nvd.nist.gov/cpe.cfm
type List struct {
	Items []Item `xml:"cpe-item"`
}

// Item has CPE information
type Item struct {
	Name      string    `xml:"name,attr"`
	Cpe23Item Cpe23Item `xml:"cpe23-item"`
	Titles    []Title   `xml:"title"`
}

// Cpe23Item : Cpe23Item
type Cpe23Item struct {
	Name string `xml:"name,attr"`
}

// Title has title, lang
type Title struct {
	Lang  string `xml:"lang,attr"`
	Value string `xml:",chardata"`
}
