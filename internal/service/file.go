package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileService представляет сервис для работы с файлами
type FileService struct {
	uploadDir string
}

// NewFileService создает новый экземпляр FileService
func NewFileService(uploadDir string) *FileService {
	return &FileService{
		uploadDir: uploadDir,
	}
}

// UploadFileParams содержит параметры для загрузки файла
type UploadFileParams struct {
	File         *multipart.FileHeader
	SubDirectory string
	FileName     string
}

// UploadFileResult содержит результат загрузки файла
type UploadFileResult struct {
	FilePath    string
	FileName    string
	FileSize    int64
	ContentType string
	UploadedAt  time.Time
}

// UploadFile загружает файл на сервер
func (s *FileService) UploadFile(ctx context.Context, params UploadFileParams) (*UploadFileResult, error) {
	// Проверяем, что файл существует
	if params.File == nil {
		return nil, errors.New("file is required")
	}

	// Открываем файл
	src, err := params.File.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Создаем директорию для загрузки, если она не существует
	uploadPath := s.uploadDir
	if params.SubDirectory != "" {
		uploadPath = filepath.Join(s.uploadDir, params.SubDirectory)
	}

	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Генерируем имя файла
	fileName := params.FileName
	if fileName == "" {
		// Используем оригинальное имя файла, но очищаем его от небезопасных символов
		originalName := params.File.Filename
		ext := filepath.Ext(originalName)
		baseName := strings.TrimSuffix(originalName, ext)
		baseName = sanitizeFileName(baseName)
		fileName = fmt.Sprintf("%s_%d%s", baseName, time.Now().Unix(), ext)
	} else {
		fileName = sanitizeFileName(fileName)
	}

	// Создаем полный путь к файлу
	filePath := filepath.Join(uploadPath, fileName)

	// Создаем файл на диске
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// Копируем содержимое файла
	fileSize, err := io.Copy(dst, src)
	if err != nil {
		// Удаляем частично загруженный файл в случае ошибки
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Получаем информацию о файле
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &UploadFileResult{
		FilePath:    filePath,
		FileName:    fileName,
		FileSize:    fileSize,
		ContentType: params.File.Header.Get("Content-Type"),
		UploadedAt:  fileInfo.ModTime(),
	}, nil
}

// DeleteFile удаляет файл с сервера
func (s *FileService) DeleteFile(ctx context.Context, filePath string) error {
	// Проверяем, что путь к файлу указан
	if filePath == "" {
		return errors.New("file path is required")
	}

	// Проверяем, что файл существует
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return errors.New("file not found")
	}

	// Удаляем файл
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// sanitizeFileName очищает имя файла от небезопасных символов
func sanitizeFileName(fileName string) string {
	// Удаляем небезопасные символы
	fileName = strings.ReplaceAll(fileName, "..", "")
	fileName = strings.ReplaceAll(fileName, "/", "_")
	fileName = strings.ReplaceAll(fileName, "\\", "_")
	fileName = strings.ReplaceAll(fileName, ":", "_")
	fileName = strings.ReplaceAll(fileName, "*", "_")
	fileName = strings.ReplaceAll(fileName, "?", "_")
	fileName = strings.ReplaceAll(fileName, "\"", "_")
	fileName = strings.ReplaceAll(fileName, "<", "_")
	fileName = strings.ReplaceAll(fileName, ">", "_")
	fileName = strings.ReplaceAll(fileName, "|", "_")

	// Удаляем начальные и конечные пробелы
	fileName = strings.TrimSpace(fileName)

	// Если имя файла пустое после очистки, используем дефолтное имя
	if fileName == "" {
		fileName = "file"
	}

	return fileName
}
