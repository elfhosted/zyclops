package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"text/template"

	"github.com/elfhosted/zyclops/pkg/domain"
	"github.com/elfhosted/zyclops/pkg/repository"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type SearchService interface {
	IndexTorrents() error
	Search(query string) ([]domain.SearchResult, error)
}

type serviceTemplateData struct {
	Name         string
	Namespace    string
	ClusterIP    string
	ExternalIP   string
	Port         string
	TargetPort   string
	NodePort     string
	LoadBalancer string
	ServiceType  string
}

type searchService struct {
	searchRepo        repository.SearchRepository
	k8sClient         *kubernetes.Clientset // can be nil
	zurgLabel         string
	zurgURLTemplate   string
	urlTemplate       *template.Template
	externalEndpoints []string
}

func NewSearchService(
	searchRepo repository.SearchRepository,
	k8sClient *kubernetes.Clientset,
	zurgLabel string,
	zurgURLTemplate string,
	externalEndpoints []string,
) SearchService {
	if zurgLabel == "" {
		zurgLabel = "app.elfhosted.com/name=zurg"
	}
	if zurgURLTemplate == "" {
		zurgURLTemplate = "http://zurg.{{.Namespace}}:9999/debug/torrents"
	}

	tmpl, err := template.New("zurgURL").Parse(zurgURLTemplate)
	if err != nil {
		// Fallback to simple template if parsing fails
		tmpl, _ = template.New("zurgURL").Parse("http://zurg.{{.Namespace}}:9999/debug/torrents")
	}

	return &searchService{
		searchRepo:        searchRepo,
		k8sClient:         k8sClient,
		zurgLabel:         zurgLabel,
		zurgURLTemplate:   zurgURLTemplate,
		urlTemplate:       tmpl,
		externalEndpoints: externalEndpoints,
	}
}

func (s *searchService) IndexTorrents() error {
	// Only process K8s services if client is available
	if s.k8sClient != nil {
		zurgs, err := s.k8sClient.CoreV1().Services("").List(context.TODO(), metav1.ListOptions{
			LabelSelector: s.zurgLabel,
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to list K8s services")
			// Don't return error, continue with external endpoints
		} else {
			log.Info().
				Int("services_found", len(zurgs.Items)).
				Str("label", s.zurgLabel).
				Msg("Found Zurg services")

			for _, svc := range zurgs.Items {
				logger := log.With().
					Str("service", svc.Name).
					Str("namespace", svc.Namespace).
					Logger()

				data := serviceTemplateData{
					Name:        svc.Name,
					Namespace:   svc.Namespace,
					ClusterIP:   svc.Spec.ClusterIP,
					ServiceType: string(svc.Spec.Type),
				}

				if len(svc.Spec.ExternalIPs) > 0 {
					data.ExternalIP = svc.Spec.ExternalIPs[0]
				}

				if len(svc.Spec.Ports) > 0 {
					data.Port = fmt.Sprintf("%d", svc.Spec.Ports[0].Port)
					if svc.Spec.Ports[0].TargetPort.IntValue() != 0 {
						data.TargetPort = fmt.Sprintf("%d", svc.Spec.Ports[0].TargetPort.IntValue())
					}
					if svc.Spec.Ports[0].NodePort != 0 {
						data.NodePort = fmt.Sprintf("%d", svc.Spec.Ports[0].NodePort)
					}
				}

				if len(svc.Status.LoadBalancer.Ingress) > 0 {
					data.LoadBalancer = svc.Status.LoadBalancer.Ingress[0].IP
					if data.LoadBalancer == "" {
						data.LoadBalancer = svc.Status.LoadBalancer.Ingress[0].Hostname
					}
				}

				var serviceURL bytes.Buffer
				if err := s.urlTemplate.Execute(&serviceURL, data); err != nil {
					logger.Error().Err(err).Msg("Failed to process URL template")
					continue
				}

				if err := s.fetchAndIndexTorrents(serviceURL.String()); err != nil {
					logger.Error().Err(err).Str("url", serviceURL.String()).Msg("Failed to fetch torrents")
				} else {
					logger.Info().Str("url", serviceURL.String()).Msg("Successfully indexed torrents")
				}
			}
		}
	} else {
		log.Info().Msg("Kubernetes client not configured, skipping service discovery")
	}

	// Then, index from external endpoints
	log.Info().
		Int("endpoints", len(s.externalEndpoints)).
		Msg("Processing external endpoints")

	for _, endpoint := range s.externalEndpoints {
		logger := log.With().Str("endpoint", endpoint).Logger()

		if err := s.fetchAndIndexTorrents(endpoint); err != nil {
			logger.Error().Err(err).Msg("Failed to fetch torrents from external endpoint")
		} else {
			logger.Info().Msg("Successfully indexed torrents from external endpoint")
		}
	}

	return nil
}

func (s *searchService) fetchAndIndexTorrents(serviceURL string) error {
	logger := log.With().Str("url", serviceURL).Logger()

	resp, err := http.Get(serviceURL)
	if err != nil {
		return fmt.Errorf("fetch error: %v", err)
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

	logger.Info().Int("count", len(torrents)).Msg("Fetched torrents")

	var indexed, skipped int
	for _, torrent := range torrents {
		if torrent.InfoHash == "" {
			logger.Warn().Str("name", torrent.Name).Msg("Skipping torrent with empty InfoHash")
			continue
		}

		exists, err := s.searchRepo.Exists(torrent.InfoHash)
		if err != nil {
			logger.Error().Err(err).Str("infohash", torrent.InfoHash).Msg("Failed to check existence")
			continue
		}

		if exists {
			skipped++
			continue
		}

		if err := s.searchRepo.Index(torrent); err != nil {
			logger.Error().
				Err(err).
				Str("torrent", torrent.Name).
				Str("infohash", torrent.InfoHash).
				Msg("Failed to index torrent")
		} else {
			indexed++
		}
	}

	logger.Info().
		Int("total", len(torrents)).
		Int("indexed", indexed).
		Int("skipped", skipped).
		Msg("Torrent indexing completed")

	return nil
}

func (s *searchService) Search(query string) ([]domain.SearchResult, error) {
	return s.searchRepo.Search(query)
}
