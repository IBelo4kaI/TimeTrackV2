package smtpsettings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

// Ключи в system_settings, category "smtp" — настраиваются на странице
// "Настройки", как и остальные (см. internal/notification, internal/vacation_type).
const (
	hostKey     = "smtp_host"
	portKey     = "smtp_port"
	usernameKey = "smtp_username"
	passwordKey = "smtp_password"
	fromKey     = "smtp_from"
)

// Config — расшифрованные настройки, для internal/mail (реальная отправка).
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// PublicSettings — то, что можно отдать через API/UI: пароль никогда не
// возвращается, ни в открытом, ни в зашифрованном виде, только флаг "задан".
type PublicSettings struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	From        string `json:"from"`
	PasswordSet bool   `json:"passwordSet"`
}

// UpdateParams — Password == nil значит "не менять пароль", как обычно у
// password-полей в формах настроек (пустая строка от фронта — не "очистить").
type UpdateParams struct {
	Host     string
	Port     int
	Username string
	From     string
	Password *string
}

type Service interface {
	// GetConfig — с расшифрованным паролем, только для internal/mail.
	GetConfig(ctx context.Context) (Config, error)
	GetPublicSettings(ctx context.Context) (PublicSettings, error)
	UpdateSettings(ctx context.Context, params UpdateParams) error
}

type service struct {
	repo          repo.Querier
	encryptionKey string
}

func NewService(r repo.Querier, encryptionKey string) Service {
	return &service{repo: r, encryptionKey: encryptionKey}
}

func (s *service) GetConfig(ctx context.Context) (Config, error) {
	host, err := s.getString(ctx, hostKey)
	if err != nil {
		return Config{}, err
	}
	portStr, err := s.getString(ctx, portKey)
	if err != nil {
		return Config{}, err
	}
	username, err := s.getString(ctx, usernameKey)
	if err != nil {
		return Config{}, err
	}
	from, err := s.getString(ctx, fromKey)
	if err != nil {
		return Config{}, err
	}
	encryptedPassword, err := s.getString(ctx, passwordKey)
	if err != nil {
		return Config{}, err
	}

	password, err := decrypt(s.encryptionKey, encryptedPassword)
	if err != nil {
		return Config{}, fmt.Errorf("decrypt smtp password: %w", err)
	}

	port, _ := strconv.Atoi(portStr)
	if port == 0 {
		port = 587
	}

	return Config{Host: host, Port: port, Username: username, Password: password, From: from}, nil
}

func (s *service) GetPublicSettings(ctx context.Context) (PublicSettings, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return PublicSettings{}, err
	}
	return PublicSettings{
		Host:        cfg.Host,
		Port:        cfg.Port,
		Username:    cfg.Username,
		From:        cfg.From,
		PasswordSet: cfg.Password != "",
	}, nil
}

func (s *service) UpdateSettings(ctx context.Context, params UpdateParams) error {
	if err := s.setString(ctx, hostKey, params.Host); err != nil {
		return err
	}
	if err := s.setString(ctx, portKey, strconv.Itoa(params.Port)); err != nil {
		return err
	}
	if err := s.setString(ctx, usernameKey, params.Username); err != nil {
		return err
	}
	if err := s.setString(ctx, fromKey, params.From); err != nil {
		return err
	}

	if params.Password != nil {
		if s.encryptionKey == "" {
			return errors.New("smtp encryption key is not configured (SMTP_ENCRYPTION_KEY)")
		}
		encrypted, err := encrypt(s.encryptionKey, *params.Password)
		if err != nil {
			return fmt.Errorf("encrypt smtp password: %w", err)
		}
		if err := s.setString(ctx, passwordKey, encrypted); err != nil {
			return err
		}
	}

	return nil
}

func (s *service) getString(ctx context.Context, key string) (string, error) {
	setting, err := s.repo.GetSystemSettingByKey(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if !setting.SettingValue.Valid {
		return "", nil
	}
	return setting.SettingValue.String, nil
}

func (s *service) setString(ctx context.Context, key, value string) error {
	return s.repo.UpdateValueSystemSetting(ctx, repo.UpdateValueSystemSettingParams{
		SettingKey:   key,
		SettingValue: sql.NullString{String: value, Valid: value != ""},
	})
}
