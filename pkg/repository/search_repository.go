package repository

import (
	"fmt"

	"github.com/blevesearch/bleve/v2"
	"github.com/elfhosted/zyclops/pkg/domain"
)

type SearchRepository interface {
	Index(torrent domain.Torrent) error
	Search(query string) ([]domain.SearchResult, error)
	Exists(infoHash string) (bool, error)
}

type bleveSearchRepository struct {
	index bleve.Index
}

func NewBleveSearchRepository(indexPath string) (SearchRepository, error) {
	// Try to open existing index first
	idx, err := bleve.Open(indexPath)
	if err != nil {
		// If index doesn't exist, create a new one
		if err == bleve.ErrorIndexPathDoesNotExist {
			idx, err = bleve.New(indexPath, bleve.NewIndexMapping())
			if err != nil {
				return nil, fmt.Errorf("failed to create new index: %v", err)
			}
		} else {
			return nil, fmt.Errorf("failed to open existing index: %v", err)
		}
	}
	return &bleveSearchRepository{index: idx}, nil
}

func (r *bleveSearchRepository) Index(torrent domain.Torrent) error {
	// Use InfoHash as the unique document ID
	return r.index.Index(torrent.InfoHash, torrent)
}

func (r *bleveSearchRepository) Search(query string) ([]domain.SearchResult, error) {
	nameQuery := bleve.NewMatchQuery(query)
	// nameQuery.SetField("name")

	searchRequest := bleve.NewSearchRequest(nameQuery)
	searchRequest.Fields = []string{"name", "hash", "size"}
	searchRequest.SortBy([]string{"-_score", "name"}) // Sort by score desc, then name
	searchRequest.Size = 10

	searchResult, err := r.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	var results []domain.SearchResult
	for _, hit := range searchResult.Hits {
		results = append(results, domain.SearchResult{
			RawTitle: hit.Fields["name"].(string),
			InfoHash: hit.Fields["hash"].(string),
			Size:     int64(hit.Fields["size"].(float64)),
		})
	}
	return results, nil
}

func (r *bleveSearchRepository) Exists(infoHash string) (bool, error) {
	doc, err := r.index.Document(infoHash)
	if err != nil {
		return false, err
	}
	return doc != nil, nil
}
