package provider

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/txomon/sawyer/pkg/util"
)

type PhotoLinker struct {
	backend        PhotoProvider
	cacheDirectory string
	memory         MapMemory
}

func (pl *PhotoLinker) Run(photoProvider *PhotoProvider) {
	var pp PhotoProvider = pl
	if photoProvider == nil {
		photoProvider = &pp
	}
	pl.backend.Run(photoProvider)
}
func (pl *PhotoLinker) SetStorageLocation(cacheDirectory string) {
	pl.cacheDirectory = cacheDirectory
	pl.memory = NewMemory(cacheDirectory)
}

func (pl *PhotoLinker) String() string {
	return fmt.Sprint("linked-", pl.GetName())
}
func (pl *PhotoLinker) GetName() string {
	return pl.backend.GetName()
}

func (pl *PhotoLinker) GetPhotos() ([]string, error) {
	logger.Tracef("Storagedirectory to %v, %p", pl.cacheDirectory, &pl)
	photos := make([]string, 0)

	backendPhotos, err := pl.backend.GetPhotos()
	if err != nil {
		logger.Infof("Failed to get photos from %v. Doing nothing", pl.backend)
		return photos, err
	}
	logger.Tracef("Getting photos and storing them in %v", pl.cacheDirectory)

	for _, backendPhotoPath := range backendPhotos {
		logger.Tracef("Procesing photo %v", backendPhotoPath)
		if cachedFile := pl.memory.getMemory(backendPhotoPath); cachedFile != "" {
			if _, err := os.Stat(cachedFile); err == nil {
				photos = append(photos, cachedFile)
				logger.Tracef("Cached file, nothing needs to be done")
				continue
			} else {
				logger.Tracef("Cached file deleted, continuing as if not cached")

			}
		}
		info, err := os.Stat(backendPhotoPath)
		if err != nil {
			logger.Infof("Could not stat file %v. %v", backendPhotoPath, err)
			continue
		} else if info.IsDir() {
			logger.Debugf("File %v is directory, skipping", backendPhotoPath)
		}

		backendPhotoFile, err := os.Open(backendPhotoPath)
		if err != nil {
			logger.Infof("Failed to open file %v", backendPhotoPath)
			continue
		}
		defer backendPhotoFile.Close()

		backendPhotoContent := make([]byte, info.Size())
		if _, err = backendPhotoFile.Read(backendPhotoContent); err != nil {
			logger.Infof("Failed to read contents from file %v", backendPhotoPath)
			continue
		}

		sum := sha1.Sum(backendPhotoContent)
		photoName := hex.EncodeToString(sum[:])
		format := util.IsBackground(false, backendPhotoPath)
		photoFileName := fmt.Sprintf("%v.%v", photoName, format)

		photoPath := filepath.Join(pl.cacheDirectory, photoFileName)

		if info, err := os.Stat(photoPath); err == nil {
			if info.IsDir() {
				logger.Warningf("The to-be-linked exists and is a directory! %v", photoPath)
			}
			logger.Debugf("File %v exists, doing nothing.", photoPath)
			photos = append(photos, photoPath)
			pl.memory.setMemory(backendPhotoPath, photoPath)
			continue
		}

		if err := os.Link(backendPhotoPath, photoPath); err != nil {
			logger.Debugf("Failed to link file, copying it. %v", err)
			if photoFile, err := os.Create(photoPath); err != nil {
				logger.Warningf("Failed to create file in cache dir %v", photoPath)
				continue
			} else {
				go func() {
					photoFile.Write(backendPhotoContent)
					photoFile.Close()
				}()
			}
		} else {
			logger.Tracef("Linked file %v to %v", backendPhotoPath, photoPath)
		}
		photos = append(photos, photoPath)
		pl.memory.setMemory(backendPhotoPath, photoPath)
	}
	return photos, nil
}
