package utils

import (
	data "finalProject/StructureData"
	"strings"
)

func ContainsInt(slice []int, value int) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func ContainsString(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func ContainsIgnoreCase(field, keyword string) bool {
	return strings.Contains(strings.ToLower(field), strings.ToLower(keyword))
}

func ContainsAnyString(slice []string, target []string) bool {
	for _, s := range slice {
		for _, t := range target {
			if s == t {
				return true
			}
		}
	}
	return false
}

func MatchAuthorCriteria(author data.Author, criteria data.AuthorSearchCriteria) bool {
	if len(criteria.IDs) > 0 && !ContainsInt(criteria.IDs, author.ID) {
		return false
	}
	if len(criteria.FirstNames) > 0 && !ContainsString(criteria.FirstNames, author.FirstName) {
		return false
	}
	if len(criteria.LastNames) > 0 && !ContainsString(criteria.LastNames, author.LastName) {
		return false
	}
	if len(criteria.Keywords) > 0 {
		for _, keyword := range criteria.Keywords {
			if ContainsIgnoreCase(author.FirstName, keyword) ||
				ContainsIgnoreCase(author.LastName, keyword) ||
				ContainsIgnoreCase(author.Bio, keyword) {
				return true
			}
		}
		return false
	}
	return true
}
