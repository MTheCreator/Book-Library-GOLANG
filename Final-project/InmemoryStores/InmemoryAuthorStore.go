package InmemoryStores

import (
	"sync"

	interfaces "finalProject/Interfaces"
	data "finalProject/StructureData"
	"finalProject/utils"
)

type InMemoryAuthorStore struct {
	mu      sync.RWMutex
	authors map[int]data.Author
	nextID  int
}

var (
	authorStoreInstance *InMemoryAuthorStore
	authorOnce          sync.Once
)

// GetAuthorStoreInstance returns the singleton instance of InMemoryAuthorStore
func GetAuthorStoreInstance() interfaces.AuthorStore {
	authorOnce.Do(func() {
		authorStoreInstance = &InMemoryAuthorStore{
			authors: make(map[int]data.Author),
			nextID:  1,
		}
	})
	return authorStoreInstance
}

// CreateAuthor adds a new author to the store
func (store *InMemoryAuthorStore) CreateAuthor(author data.Author) (data.Author, *data.ErrorResponse) {
	store.mu.Lock()
	defer store.mu.Unlock()

	author.ID = store.nextID
	store.nextID++
	store.authors[author.ID] = author
	return author, nil
}

// GetAuthor retrieves an author by ID
func (store *InMemoryAuthorStore) GetAuthor(id int) (data.Author, *data.ErrorResponse) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	author, exists := store.authors[id]
	if !exists {
		return data.Author{}, &data.ErrorResponse{Message: "Author not found"}
	}
	return author, nil
}

// GetAllAuthors retrieves all authors from the store
func (store *InMemoryAuthorStore) GetAllAuthors() []data.Author {
	store.mu.RLock()
	defer store.mu.RUnlock()

	var authors []data.Author
	for _, author := range store.authors {
		authors = append(authors, author)
	}
	return authors
}

// UpdateAuthor updates an author's details
func (store *InMemoryAuthorStore) UpdateAuthor(id int, author data.Author) (data.Author, *data.ErrorResponse) {
	store.mu.Lock()
	defer store.mu.Unlock()

	_, exists := store.authors[id]
	if !exists {
		return data.Author{}, &data.ErrorResponse{Message: "Author not found"}
	}
	author.ID = id
	store.authors[id] = author
	return author, nil
}

// DeleteAuthor removes an author by ID
func (store *InMemoryAuthorStore) DeleteAuthor(id int) *data.ErrorResponse {
	store.mu.Lock()
	defer store.mu.Unlock()

	_, exists := store.authors[id]
	if !exists {
		return &data.ErrorResponse{Message: "Author not found"}
	}
	delete(store.authors, id)
	return nil
}

// SearchAuthors filters authors based on the search criteria
func (store *InMemoryAuthorStore) SearchAuthors(criteria data.AuthorSearchCriteria) ([]data.Author, *data.ErrorResponse) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	var result []data.Author
	for _, author := range store.authors {
		if len(criteria.IDs) > 0 && !utils.ContainsInt(criteria.IDs, author.ID) {
			continue
		}
		if len(criteria.FirstNames) > 0 && !utils.ContainsString(criteria.FirstNames, author.FirstName) {
			continue
		}
		if len(criteria.LastNames) > 0 && !utils.ContainsString(criteria.LastNames, author.LastName) {
			continue
		}
		if len(criteria.Keywords) > 0 {
			matched := false
			for _, keyword := range criteria.Keywords {
				if utils.ContainsIgnoreCase(author.FirstName, keyword) ||
					utils.ContainsIgnoreCase(author.LastName, keyword) ||
					utils.ContainsIgnoreCase(author.Bio, keyword) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		result = append(result, author)
	}
	return result, nil
}
