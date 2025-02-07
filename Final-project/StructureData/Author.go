package StructureData

type Author struct {
	ID int `json:"id"`
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
	Bio string `json:"bio"`
   }
   
   type AuthorSearchCriteria struct {
	IDs         []int    `json:"ids,omitempty"`         
	FirstNames  []string `json:"first_names,omitempty"`  
	LastNames   []string `json:"last_names,omitempty"`   
	Keywords    []string `json:"keywords,omitempty"`     
}