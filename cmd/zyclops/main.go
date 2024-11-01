package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/elfhosted/zyclops/pkg/handler"
	"github.com/elfhosted/zyclops/pkg/repository"
	"github.com/elfhosted/zyclops/pkg/service"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Config struct {
	KubeconfigPath string
	IndexPath      string
	ServerPort     int
	ServerHost     string
	SearchEndpoint string
}

func loadConfig() Config {
	var config Config

	// Load configuration from environment variables with defaults
	config.KubeconfigPath = getEnv("KUBECONFIG", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	config.IndexPath = getEnv("INDEX_PATH", "torrents.bleve")
	config.ServerPort = getEnvAsInt("SERVER_PORT", 8080)
	config.ServerHost = getEnv("SERVER_HOST", "")
	config.SearchEndpoint = getEnv("SEARCH_ENDPOINT", "/dmm/search")

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

func main() {
	config := loadConfig()

	// Initialize Kubernetes client
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", config.KubeconfigPath)
	if err != nil {
		fmt.Printf("Error building kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		fmt.Printf("Error creating Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	// Initialize repository
	searchRepo, err := repository.NewBleveSearchRepository(config.IndexPath)
	if err != nil {
		fmt.Printf("Error creating Bleve index: %v\n", err)
		os.Exit(1)
	}

	// Initialize service
	searchService := service.NewSearchService(searchRepo, clientset)

	// Initialize handler
	searchHandler := handler.NewSearchHandler(searchService)

	// Index initial torrents
	if err := searchService.IndexTorrents(); err != nil {
		fmt.Printf("Error indexing torrents: %v\n", err)
		os.Exit(1)
	}

	// Setup HTTP server
	http.HandleFunc(config.SearchEndpoint, searchHandler.Search)
	addr := fmt.Sprintf("%s:%d", config.ServerHost, config.ServerPort)
	fmt.Printf("Server started at %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
		os.Exit(1)
	}
}
