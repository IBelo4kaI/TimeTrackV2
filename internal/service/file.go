package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
	"timetrack/internal/adapter/storage"
	repo "timetrack/internal/adapter/mysql/sqlc"

	"github.com/google/uuid"
)

var ErrFileNotFound = errors.New("file not found")

type FileService struct {
	repo    *repo.Queries
	db      *sql.DB
	storage *storage.DiskStorage
}

func NewFileService(db *sql.DB, basePath string) *FileService {
	return &FileService{
		repo:    repo.New(db),
		db:      db,
		storage: storage.NewDiskStorage(basePath),
	}
}

type UploadFileParams struct {
	File       *multipart.FileHeader
	EntityType string
	EntityID   string
	UploaderID string
}

func (s *FileService) Upload(ctx context.Context, p UploadFileParams) (repo.File, error) {
	if p.File == nil {
		return repo.File{}, errors.New("file is required")
	}

	src, err := p.File.Open()
	if err != nil {
		return repo.File{}, fmt.Errorf("open file: %w", err)
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return repo.File{}, fmt.Errorf("read file: %w", err)
	}

	fileID := uuid.NewString()
	storagePath, checksum, err := s.storage.Save(fileID, p.File.Filename, data)
	if err != nil {
		return repo.File{}, fmt.Errorf("save to disk: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.storage.Delete(storagePath)
		return repo.File{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)

	if err = qtx.CreateFile(ctx, repo.CreateFileParams{
		ID:               fileID,
		OriginalName:     filepath.Base(p.File.Filename),
		StoragePath:      storagePath,
		MimeType:         p.File.Header.Get("Content-Type"),
		FileType:         detectFileType(p.File.Header.Get("Content-Type")),
		SizeBytes:        p.File.Size,
		Checksum:         checksum,
		UploadedByUserID: p.UploaderID,
	}); err != nil {
		s.storage.Delete(storagePath)
		return repo.File{}, fmt.Errorf("create file record: %w", err)
	}

	if p.EntityType != "" && p.EntityID != "" {
		if err = qtx.CreateFileEntityRef(ctx, repo.CreateFileEntityRefParams{
			FileID:     fileID,
			EntityType: p.EntityType,
			EntityID:   p.EntityID,
		}); err != nil {
			s.storage.Delete(storagePath)
			return repo.File{}, fmt.Errorf("create entity ref: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		s.storage.Delete(storagePath)
		return repo.File{}, fmt.Errorf("commit tx: %w", err)
	}

	f, err := s.repo.GetFileByID(ctx, fileID)
	if err != nil {
		return repo.File{}, fmt.Errorf("get uploaded file: %w", err)
	}
	return f, nil
}

func (s *FileService) GetFile(ctx context.Context, id string) (repo.File, error) {
	f, err := s.repo.GetFileByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repo.File{}, ErrFileNotFound
		}
		return repo.File{}, fmt.Errorf("get file: %w", err)
	}
	return f, nil
}

func (s *FileService) Delete(ctx context.Context, id string) error {
	f, err := s.repo.GetFileByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFileNotFound
		}
		return fmt.Errorf("get file: %w", err)
	}

	if err = s.repo.HardDeleteFile(ctx, id); err != nil {
		return fmt.Errorf("hard delete: %w", err)
	}

	_ = s.storage.Delete(f.StoragePath)
	return nil
}

func (s *FileService) ListByEntity(ctx context.Context, entityType, entityID string) ([]repo.File, error) {
	files, err := s.repo.ListFilesByEntity(ctx, repo.ListFilesByEntityParams{
		EntityType: entityType,
		EntityID:   entityID,
	})
	if err != nil {
		return nil, fmt.Errorf("list files by entity: %w", err)
	}
	return files, nil
}

// --- backward-compat API used by VacationHandler ---

type LegacyUploadFileParams struct {
	File         *multipart.FileHeader
	SubDirectory string
	FileName     string
}

type LegacyUploadFileResult struct {
	FilePath    string
	FileName    string
	FileSize    int64
	ContentType string
	UploadedAt  time.Time
}

func (s *FileService) UploadFile(ctx context.Context, params LegacyUploadFileParams) (*LegacyUploadFileResult, error) {
	if params.File == nil {
		return nil, errors.New("file is required")
	}

	src, err := params.File.Open()
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer src.Close()

	uploadPath := s.storage.BasePath()
	if params.SubDirectory != "" {
		uploadPath = filepath.Join(uploadPath, params.SubDirectory)
	}

	if err = os.MkdirAll(uploadPath, 0755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}

	fileName := params.FileName
	if fileName == "" {
		orig := params.File.Filename
		ext := filepath.Ext(orig)
		base := sanitizeFileName(strings.TrimSuffix(orig, ext))
		fileName = fmt.Sprintf("%s_%d%s", base, time.Now().Unix(), ext)
	} else {
		fileName = sanitizeFileName(fileName)
	}

	filePath := filepath.Join(uploadPath, fileName)
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	size, err := io.Copy(dst, src)
	if err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("save file: %w", err)
	}

	fi, _ := os.Stat(filePath)
	uploadedAt := time.Now()
	if fi != nil {
		uploadedAt = fi.ModTime()
	}

	return &LegacyUploadFileResult{
		FilePath:    filePath,
		FileName:    fileName,
		FileSize:    size,
		ContentType: params.File.Header.Get("Content-Type"),
		UploadedAt:  uploadedAt,
	}, nil
}

func (s *FileService) DeleteFile(ctx context.Context, filePath string) error {
	if filePath == "" {
		return errors.New("file path is required")
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return errors.New("file not found")
	}
	return os.Remove(filePath)
}

func sanitizeFileName(name string) string {
	for _, ch := range []string{"..", "/", "\\", ":", "*", "?", "\"", "<", ">", "|"} {
		name = strings.ReplaceAll(name, ch, "_")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "file"
	}
	return name
}

// --- helpers ---

func detectFileType(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case mimeType == "application/pdf":
		return "document"
	default:
		return "other"
	}
}
