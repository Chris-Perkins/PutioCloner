package cloning

import (
	"context"
	"path/filepath"
	"putio_cloner/registry"

	"github.com/putdotio/go-putio/putio"
)

const (
	rootFolderId          int64 = 0
	makeDirPermissionMode       = 0755
)

// PutioScanner is used to scan for new items available in Putio. New items
// are requested for download to the injected DownloadManager.
type PutioScanner struct {
	client          *putio.Client
	registry        *registry.StringRegistry
	downloadManager *DownloadManager
	chunkSize       int
	maxThreads      int
}

func NewPutioScanner(client *putio.Client, registry *registry.StringRegistry, downloadManager *DownloadManager) *PutioScanner {
	return &PutioScanner{
		client:          client,
		registry:        registry,
		downloadManager: downloadManager,
	}
}

func (c *PutioScanner) CopyNewItemsToFolder(ctx context.Context, path string) error {
	rootFolderContents, _, err := c.client.Files.List(ctx, rootFolderId)
	if err != nil {
		return err
	}
	for _, f := range rootFolderContents {
		err := c.recursivelyCopyNewData(ctx, f, path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *PutioScanner) recursivelyCopyNewData(ctx context.Context, file putio.File, outPath string) error {
	registryKey := getRegistryKey(file)
	if c.registry.IsRegistered(registryKey) {
		return nil
	}

	var err error
	if file.IsDir() {
		err = c.handleDirectory(ctx, file, outPath)
	} else {
		err = c.downloadItem(ctx, file, outPath)
	}

	if err == nil {
		c.registry.Register(registryKey)
	}

	return err
}

func (c *PutioScanner) handleDirectory(ctx context.Context, dir putio.File, outPath string) error {
	files, _, err := c.client.Files.List(ctx, dir.ID)

	directoryPath := filepath.Join(outPath, dir.Name, "")
	if err != nil {
		return err
	}

	for _, f := range files {
		err := c.recursivelyCopyNewData(ctx, f, directoryPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *PutioScanner) downloadItem(ctx context.Context, file putio.File, downloadPath string) error {
	url, err := c.client.Files.URL(ctx, file.ID, false)
	if err != nil {
		return err
	}
	return c.downloadManager.RequestDownload(url, filepath.Join(downloadPath, file.Name))
}

func getRegistryKey(file putio.File) string {
	return file.CRC32 + file.UpdatedAt.String()
}
