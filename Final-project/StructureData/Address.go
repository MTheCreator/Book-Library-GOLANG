package StructureData

type Address struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type AddressSearchCriteria struct {
	Streets     []string `json:"streets,omitempty"`       // Filter by street names
	Cities      []string `json:"cities,omitempty"`        // Filter by city names
	States      []string `json:"states,omitempty"`        // Filter by states
	PostalCodes []string `json:"postal_codes,omitempty"`  // Filter by postal codes
	Countries   []string `json:"countries,omitempty"`     // Filter by countries
}
