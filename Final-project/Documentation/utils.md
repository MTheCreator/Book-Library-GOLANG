# Project Documentation (Deprecated)

## utilFunctions.go

This file contains utility functions that provide common operations for handling data structures and search criteria.

### Utility Functions

#### ContainsInt
Checks if a specific integer exists in a slice of integers.
```go
func ContainsInt(slice []int, value int) bool
```

#### ContainsString
Checks if a specific string exists in a slice of strings.
```go
func ContainsString(slice []string, value string) bool
```

#### ContainsIgnoreCase
Performs a case-insensitive check to see if a keyword exists in a given string field.
```go
func ContainsIgnoreCase(field, keyword string) bool
```

#### ContainsAnyString
Checks if any string from a target slice exists in a source slice.
```go
func ContainsAnyString(slice []string, target []string) bool
```

#### MatchAuthorCriteria
Checks if an author matches the provided search criteria.
```go
func MatchAuthorCriteria(author data.Author, criteria data.AuthorSearchCriteria) bool
```
- **Criteria Evaluated**:
  - Author ID matches one of the specified IDs.
  - Author's first name or last name matches specified names.
  - Author's bio contains specified keywords (case-insensitive).
