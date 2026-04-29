package storage

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

type DiskStorage struct {
	basePath string
}

func NewDiskStorage(basePath string) *DiskStorage {
	return &DiskStorage{basePath: basePath}
}

func (s *DiskStorage) BasePath() string {
	return s.basePath
}

// Save записывает данные на диск и возвращает путь к файлу и SHA-256 checksum.
func (s *DiskStorage) Save(id, filename string, data []byte) (storagePath, checksum string, err error) {
	if err = os.MkdirAll(s.basePath, 0755); err != nil {
		return "", "", fmt.Errorf("create storage dir: %w", err)
	}

	ext := filepath.Ext(filename)
	storagePath = filepath.Join(s.basePath, id+ext)

	if err = os.WriteFile(storagePath, data, 0644); err != nil {
		return "", "", fmt.Errorf("write file: %w", err)
	}

	h := sha256.Sum256(data)
	checksum = fmt.Sprintf("%x", h)
	return storagePath, checksum, nil
}

// Delete удаляет файл с диска. Не возвращает ошибку если файл уже отсутствует.
func (s *DiskStorage) Delete(storagePath string) error {
	err := os.Remove(storagePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}
