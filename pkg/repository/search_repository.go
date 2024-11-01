package repository

import (
	"github.com/blevesearch/bleve/v2"
	index "github.com/blevesearch/bleve_index_api"
	"github.com/elfhosted/zyclops/pkg/domain"
)

type SearchRepository interface {
	Index(torrent domain.Torrent) error
	Search(query string) ([]domain.SearchResult, error)
}

type bleveSearchRepository struct {
	index bleve.Index
}

func NewBleveSearchRepository(indexPath string) (SearchRepository, error) {
	idx, err := bleve.New(indexPath, bleve.NewIndexMapping())
	if err != nil {
		return nil, err
	}
	return &bleveSearchRepository{index: idx}, nil
}

func (r *bleveSearchRepository) Index(torrent domain.Torrent) error {
	return r.index.Index(torrent.Hash, torrent)
}

func (r *bleveSearchRepository) Search(query string) ([]domain.SearchResult, error) {
	bleveQuery := bleve.NewQueryStringQuery(query)
	searchRequest := bleve.NewSearchRequest(bleveQuery)
	searchResult, err := r.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	var results []domain.SearchResult
	for _, hit := range searchResult.Hits {
		doc, err := r.index.Document(hit.ID)
		if err != nil {
			continue
		}

		var torrent domain.Torrent
		doc.VisitFields(func(field index.Field) {
			switch field.Name() {
			case "name":
				torrent.Name = string(field.Value())
			case "hash":
				torrent.Hash = string(field.Value())
			case "size":
				torrent.Size = int64(field.NumPlainTextBytes())
			}
		})

		results = append(results, domain.SearchResult{
			RawTitle: torrent.Name,
			InfoHash: torrent.Hash,
			Size:     torrent.Size,
		})
	}
	return results, nil
}
