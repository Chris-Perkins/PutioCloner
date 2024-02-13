package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"putio_cloner/cloning"
	"putio_cloner/registry"
	"sync"
	"time"

	"github.com/putdotio/go-putio/putio"
	"golang.org/x/oauth2"
)

const (
	putioTokenKey           string = "putio-token"
	outputFolderKey         string = "out"
	refreshRateKey          string = "refresh-rate"
	registryPathKey         string = "registry"
	downloadRequestsPathKey string = "requests"
	chunkSizeKey            string = "chunk-size"
	maxThreadsKey           string = "max-threads"
)

type configuration struct {
	putioToken           string
	outputFolder         string
	registryPath         string
	downloadRequestsPath string
	refreshRateSeconds   int
	chunkSize            int
	maxThreads           int
}

func main() {
	config := parseLaunchConfiguration()
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.putioToken},
	)
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	putioClient := putio.NewClient(oauthClient)
	registry := registry.NewStringRegistry(config.registryPath)
	downloadManager := cloning.NewDownloadManager(config.downloadRequestsPath, config.maxThreads, config.chunkSize)
	cloner := cloning.NewPutioScanner(putioClient, registry, downloadManager)

	wg := sync.WaitGroup{}

	ctx := context.Background()
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			err := cloner.CopyNewItemsToFolder(ctx, config.outputFolder)
			if err != nil {
				fmt.Println(err)
			}
			time.Sleep(time.Second * time.Duration(config.refreshRateSeconds))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			err := downloadManager.PerformDownloads()
			if err != nil {
				fmt.Println(err)
			}
			time.Sleep(time.Second * time.Duration(config.refreshRateSeconds))
		}
	}()

	wg.Wait()
}

func parseLaunchConfiguration() *configuration {
	var putioToken string
	var outputFolder string
	var registryPath string
	var requestsPath string
	var refreshRate int
	var chunkSize int
	var maxThreads int

	defaultOutput, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	flag.StringVar(&putioToken, putioTokenKey, "", "OAuth2 token for putio")
	flag.StringVar(&outputFolder, outputFolderKey, filepath.Join(defaultOutput, "Downloads"), "The download location for putio files")
	flag.StringVar(&registryPath, registryPathKey, ".registry", "The location of the local file registry")
	flag.StringVar(&requestsPath, downloadRequestsPathKey, ".requests", "The location of pending download requests")
	flag.IntVar(&refreshRate, refreshRateKey, 60, "How often this application should run its loops in seconds")
	flag.IntVar(&chunkSize, chunkSizeKey, 5*1024*1024, "The chunk size to download. Default 5 MB")
	flag.IntVar(&maxThreads, maxThreadsKey, 3, "The maximum number of threads for downloading at the directory level")
	flag.Parse()

	return &configuration{
		putioToken:           putioToken,
		outputFolder:         outputFolder,
		refreshRateSeconds:   refreshRate,
		registryPath:         registryPath,
		downloadRequestsPath: requestsPath,
		chunkSize:            chunkSize,
		maxThreads:           maxThreads,
	}
}
