package domain

type Torrent struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

type SearchRequest struct {
	QueryText string `json:"queryText"`
}

type SearchResult struct {
	RawTitle string `json:"raw_title"`
	InfoHash string `json:"info_hash"`
	Size     int64  `json:"size"`
}
