package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/elfhosted/zyclops/pkg/handler"
	"github.com/elfhosted/zyclops/pkg/repository"
	"github.com/elfhosted/zyclops/pkg/service"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Config struct {
	KubeconfigPath    string
	IndexPath         string
	ServerPort        int
	ServerHost        string
	SearchEndpoint    string
	ZurgLabel         string
	ZurgURLTemplate   string
	ExternalEndpoints []string
	HealthEndpoint    string
}

func loadConfig() Config {
	var config Config

	// Make kubeconfig optional by checking if file exists
	kubePath := getEnv("KUBECONFIG", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if _, err := os.Stat(kubePath); err == nil {
		config.KubeconfigPath = kubePath
	}

	// Load configuration from environment variables with defaults
	config.IndexPath = getEnv("INDEX_PATH", "torrents.bleve")
	config.ServerPort = getEnvAsInt("SERVER_PORT", 8080)
	config.ServerHost = getEnv("SERVER_HOST", "")
	config.SearchEndpoint = getEnv("SEARCH_ENDPOINT", "/dmm/search")
	config.ZurgLabel = getEnv("ZURG_LABEL", "app.elfhosted.com/name=zurg")
	config.ZurgURLTemplate = getEnv("ZURG_URL_TEMPLATE", "http://zurg.%s:9999/debug/torrents")

	// Load comma-separated list of external endpoints
	endpoints := getEnv("EXTERNAL_ENDPOINTS", "")
	if endpoints != "" {
		config.ExternalEndpoints = strings.Split(endpoints, ",")
	}

	config.HealthEndpoint = getEnv("HEALTH_ENDPOINT", "/health")

	return config
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func init() {
	// Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Info().Msg("No .env file found, using defaults")
	}
}

func main() {
	config := loadConfig()
	log.Info().
		Str("host", config.ServerHost).
		Int("port", config.ServerPort).
		Str("endpoint", config.SearchEndpoint).
		Int("external_endpoints", len(config.ExternalEndpoints)).
		Msg("Starting zyclops")

	// Initialize Kubernetes client if kubeconfig exists
	var clientset *kubernetes.Clientset
	if config.KubeconfigPath != "" {
		k8sConfig, err := clientcmd.BuildConfigFromFlags("", config.KubeconfigPath)
		if err != nil {
			log.Warn().
				Err(err).
				Str("path", config.KubeconfigPath).
				Msg("Failed to build kubeconfig, continuing without K8s integration")
		} else {
			clientset, err = kubernetes.NewForConfig(k8sConfig)
			if err != nil {
				log.Warn().
					Err(err).
					Msg("Error creating Kubernetes client, continuing without K8s integration")
			}
		}
	} else {
		log.Info().Msg("No kubeconfig specified, running without K8s integration")
	}

	// Initialize repository
	searchRepo, err := repository.NewBleveSearchRepository(config.IndexPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating Bleve index")
	}

	// Initialize service
	searchService := service.NewSearchService(
		searchRepo,
		clientset, // can be nil
		config.ZurgLabel,
		config.ZurgURLTemplate,
		config.ExternalEndpoints,
	)

	// Initialize handlers
	searchHandler := handler.NewSearchHandler(searchService)
	healthHandler := handler.NewHealthHandler()

	// Index initial torrents
	if err := searchService.IndexTorrents(); err != nil {
		log.Fatal().Err(err).Msg("Error indexing torrents")
	}

	// Setup HTTP server
	http.HandleFunc(config.SearchEndpoint, searchHandler.Search)
	http.HandleFunc(config.HealthEndpoint, healthHandler.Health)
	addr := fmt.Sprintf("%s:%d", config.ServerHost, config.ServerPort)
	log.Info().Str("addr", addr).Msg("Server started")
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal().Err(err).Msg("Server failed")
	}
}
