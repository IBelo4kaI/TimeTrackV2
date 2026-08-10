package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/adapter/storage"

	"github.com/google/uuid"
)

var ErrFileNotFound = errors.New("file not found")

// entityCategorySystemName — раскладка файлов по системным категориям
// (см. миграцию 010_file_category_system_name.sql) в зависимости от
// entity_type, к которому файл привязывается при загрузке. Если категория
// явно не передана в UploadFileParams.CategoryID, используется эта карта.
var entityCategorySystemName = map[string]string{
	"vacation":   "application",
	"sick_leave": "medical",
}

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
	CategoryID string
	UploaderID string
}

func (s *FileService) Upload(ctx context.Context, p UploadFileParams) (repo.GetFileByIDRow, error) {
	if p.File == nil {
		return repo.GetFileByIDRow{}, errors.New("file is required")
	}

	src, err := p.File.Open()
	if err != nil {
		return repo.GetFileByIDRow{}, fmt.Errorf("open file: %w", err)
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return repo.GetFileByIDRow{}, fmt.Errorf("read file: %w", err)
	}

	fileID := uuid.NewString()
	storagePath, checksum, err := s.storage.Save(fileID, p.File.Filename, data)
	if err != nil {
		return repo.GetFileByIDRow{}, fmt.Errorf("save to disk: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.storage.Delete(storagePath)
		return repo.GetFileByIDRow{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)

	categoryID := s.resolveCategoryID(ctx, p.CategoryID, p.EntityType)

	if err = qtx.CreateFile(ctx, repo.CreateFileParams{
		ID:               fileID,
		OriginalName:     filepath.Base(p.File.Filename),
		StoragePath:      storagePath,
		MimeType:         p.File.Header.Get("Content-Type"),
		FileType:         detectFileType(p.File.Header.Get("Content-Type")),
		CategoryID:       categoryID,
		SizeBytes:        p.File.Size,
		Checksum:         checksum,
		UploadedByUserID: p.UploaderID,
	}); err != nil {
		s.storage.Delete(storagePath)
		return repo.GetFileByIDRow{}, fmt.Errorf("create file record: %w", err)
	}

	if p.EntityType != "" && p.EntityID != "" {
		if err = qtx.CreateFileEntityRef(ctx, repo.CreateFileEntityRefParams{
			FileID:     fileID,
			EntityType: p.EntityType,
			EntityID:   p.EntityID,
		}); err != nil {
			s.storage.Delete(storagePath)
			return repo.GetFileByIDRow{}, fmt.Errorf("create entity ref: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		s.storage.Delete(storagePath)
		return repo.GetFileByIDRow{}, fmt.Errorf("commit tx: %w", err)
	}

	f, err := s.repo.GetFileByID(ctx, fileID)
	if err != nil {
		return repo.GetFileByIDRow{}, fmt.Errorf("get uploaded file: %w", err)
	}
	return f, nil
}

func (s *FileService) GetFile(ctx context.Context, id string) (repo.GetFileByIDRow, error) {
	f, err := s.repo.GetFileByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repo.GetFileByIDRow{}, ErrFileNotFound
		}
		return repo.GetFileByIDRow{}, fmt.Errorf("get file: %w", err)
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

func (s *FileService) ListByEntity(ctx context.Context, entityType, entityID string) ([]repo.ListFilesByEntityRow, error) {
	files, err := s.repo.ListFilesByEntity(ctx, repo.ListFilesByEntityParams{
		EntityType: entityType,
		EntityID:   entityID,
	})
	if err != nil {
		return nil, fmt.Errorf("list files by entity: %w", err)
	}
	return files, nil
}

func (s *FileService) ListByEntityType(ctx context.Context, entityType string) ([]repo.ListFilesByEntityTypeRow, error) {
	files, err := s.repo.ListFilesByEntityType(ctx, entityType)
	if err != nil {
		return nil, fmt.Errorf("list files by entity: %w", err)
	}
	return files, nil
}

// ListByCategory возвращает файлы, привязанные к указанной категории.
func (s *FileService) ListByCategory(ctx context.Context, categoryID string) ([]repo.ListFilesByCategoryRow, error) {
	files, err := s.repo.ListFilesByCategory(ctx, sql.NullString{String: categoryID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("list files by category: %w", err)
	}
	return files, nil
}

// SetCategory перемещает файл в другую категорию (или убирает из категории, если categoryID пустой).
func (s *FileService) SetCategory(ctx context.Context, fileID, categoryID string) error {
	if _, err := s.repo.GetFileByID(ctx, fileID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFileNotFound
		}
		return fmt.Errorf("get file: %w", err)
	}

	var category sql.NullString
	if categoryID != "" {
		category = sql.NullString{String: categoryID, Valid: true}
	}

	if err := s.repo.UpdateFileCategoryAssignment(ctx, repo.UpdateFileCategoryAssignmentParams{
		CategoryID: category,
		ID:         fileID,
	}); err != nil {
		return fmt.Errorf("update file category: %w", err)
	}
	return nil
}

// resolveCategoryID возвращает категорию для нового файла: явно переданная
// категория в приоритете, иначе — системная категория, сопоставленная с
// entityType (entityCategorySystemName). Если ни то ни другое не подошло
// (например, миграция ещё не накатилась или entityType не размечен),
// файл создаётся без категории — это не фатально.
func (s *FileService) resolveCategoryID(ctx context.Context, explicitCategoryID, entityType string) sql.NullString {
	if explicitCategoryID != "" {
		return sql.NullString{String: explicitCategoryID, Valid: true}
	}

	systemName, ok := entityCategorySystemName[entityType]
	if !ok {
		return sql.NullString{}
	}

	category, err := s.repo.GetFileCategoryBySystemName(ctx, sql.NullString{String: systemName, Valid: true})
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: category.ID, Valid: true}
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
