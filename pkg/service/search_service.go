package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/elfhosted/zyclops/pkg/domain"
	"github.com/elfhosted/zyclops/pkg/repository"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type SearchService interface {
	IndexTorrents() error
	Search(query string) ([]domain.SearchResult, error)
}

type searchService struct {
	searchRepo repository.SearchRepository
	k8sClient  *kubernetes.Clientset
}

func NewSearchService(searchRepo repository.SearchRepository, k8sClient *kubernetes.Clientset) SearchService {
	return &searchService{
		searchRepo: searchRepo,
		k8sClient:  k8sClient,
	}
}

func (s *searchService) IndexTorrents() error {
	zurgs, err := s.k8sClient.CoreV1().Services("").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app.elfhosted.com/name=zurg",
	})
	if err != nil {
		return fmt.Errorf("error listing services: %v", err)
	}

	for _, svc := range zurgs.Items {
		serviceURL := fmt.Sprintf("http://zurg.%s:9999/debug/torrents", svc.Namespace)
		if err := s.fetchAndIndexTorrents(serviceURL); err != nil {
			fmt.Printf("Error processing %s: %v\n", serviceURL, err)
		}
	}
	return nil
}

func (s *searchService) fetchAndIndexTorrents(serviceURL string) error {
	resp, err := http.Get(serviceURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var torrents []domain.Torrent
	if err := json.Unmarshal(body, &torrents); err != nil {
		return err
	}

	for _, torrent := range torrents {
		if err := s.searchRepo.Index(torrent); err != nil {
			fmt.Printf("Error indexing torrent %s: %v\n", torrent.Name, err)
		}
	}
	return nil
}

func (s *searchService) Search(query string) ([]domain.SearchResult, error) {
	return s.searchRepo.Search(query)
}
