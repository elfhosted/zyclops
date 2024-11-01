package domain

type Torrent struct {
	Name     string `json:"name"`
	InfoHash string `json:"hash"` // This is used as the unique identifier
	Size     int64  `json:"size"`
}

type SearchRequest struct {
	QueryText string `json:"queryText"`
}

type SearchResult struct {
	RawTitle string `json:"raw_title"`
	InfoHash string `json:"info_hash"`
	Size     int64  `json:"size"`
}
