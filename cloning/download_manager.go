package cloning

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"golang.org/x/sync/errgroup"
)

const (
	// DefaultFileMode represents the default file permission mode
	defaultFileMode int = 0644
)

// DownloadManager is used to handle downloading. It is able to handle multiple
// downloads at the same time.
type DownloadManager struct {
	maxThreads      int
	chunkSize       int
	persistencePath string
	storeMutex      sync.Mutex
	downloadMutex   sync.Mutex
}

func NewDownloadManager(persistencePath string, maxThreads int, chunkSize int) *DownloadManager {
	return &DownloadManager{
		maxThreads:      maxThreads,
		chunkSize:       chunkSize,
		persistencePath: persistencePath,
	}
}

type downloadRequest struct {
	DownloadURL  string `json:"downloadUrl"`
	DownloadPath string `json:"downloadPath"`
}

func (dm *DownloadManager) RequestDownload(url string, path string) error {
	return dm.addRequest(downloadRequest{
		DownloadURL:  url,
		DownloadPath: path,
	})
}

func (dm *DownloadManager) PerformDownloads() error {
	dm.downloadMutex.Lock()
	defer dm.downloadMutex.Unlock()

	requests, err := readRequestsFromFile(dm.persistencePath)
	if err != nil {
		return err
	}

	eg := errgroup.Group{}
	eg.SetLimit(dm.maxThreads)
	for _, req := range requests {
		req1 := req
		eg.Go(func() error {
			fmt.Println(fmt.Sprintf("Downloading %s", req1.DownloadPath))
			err := downloadFileInChunks(req1.DownloadURL, req1.DownloadPath, dm.chunkSize)
			if err != nil {
				return err
			}
			fmt.Println(fmt.Sprintf("- Downloaded %s", req1.DownloadPath))
			return dm.deleteRequestFromFile(req1)
		})
	}
	return eg.Wait()
}

func (dm *DownloadManager) addRequest(newRequest downloadRequest) error {
	dm.storeMutex.Lock()
	defer dm.storeMutex.Unlock()

	existingRequests, err := readRequestsFromFile(dm.persistencePath)
	if err != nil {
		return err
	}
	existingRequests = append(existingRequests, newRequest)
	err = writeRequestsToFile(dm.persistencePath, existingRequests)
	if err != nil {
		return err
	}

	return nil
}

func (dm *DownloadManager) deleteRequestFromFile(reqToDelete downloadRequest) error {
	dm.storeMutex.Lock()
	dm.storeMutex.Unlock()

	existingRequests, err := readRequestsFromFile(dm.persistencePath)
	if err != nil {
		return err
	}

	for i, req := range existingRequests {
		if req == reqToDelete {
			existingRequests = append(existingRequests[:i], existingRequests[i+1:]...)
			return writeRequestsToFile(dm.persistencePath, existingRequests)
		}
	}
	return fmt.Errorf("No request was found for url %s and path %s", reqToDelete.DownloadURL, reqToDelete.DownloadPath)
}

func readRequestsFromFile(filename string) ([]downloadRequest, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return []downloadRequest{}, nil
	}

	fileData, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var readData []downloadRequest
	err = json.Unmarshal(fileData, &readData)
	if err != nil {
		return nil, err
	}

	return readData, nil
}

func writeRequestsToFile(filename string, data []downloadRequest) error {
	jsonData, err := json.MarshalIndent(data, "", "")
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, jsonData, os.FileMode(defaultFileMode))
	if err != nil {
		return err
	}
	return nil
}

func downloadFileInChunks(url, downloadPath string, chunkSize int) error {
	// not good practice to make the dirs and whatever here but this is a side project with n=?1? users
	err := os.MkdirAll(filepath.Dir(downloadPath), makeDirPermissionMode)
	if err != nil {
		return err
	}

	file, err := os.Create(downloadPath)
	if err != nil {
		return err
	}
	defer file.Close()

	headResp, err := http.Head(url)
	if err != nil {
		return err
	}
	defer headResp.Body.Close()

	if headResp.Header.Get("Accept-Ranges") != "bytes" {
		return fmt.Errorf("Server does not support range requests")
	}

	contentLength, err := strconv.Atoi(headResp.Header.Get("Content-Length"))
	if err != nil {
		return err
	}

	for i := 0; i < contentLength; i += chunkSize {
		end := min(i+chunkSize-1, contentLength-1)

		rangeHeader := fmt.Sprintf("bytes=%d-%d", i, end)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Range", rangeHeader)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return err
		}
		// Prematurely close, otherwise we keep open gigs of data
		resp.Body.Close()
	}
	return nil
}
