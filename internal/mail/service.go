package mail

import (
	"context"
	"errors"
	"fmt"
	"io"
	smtpsettings "timetrack/internal/smtp_settings"

	gomail "gopkg.in/gomail.v2"
)

// Attachment — вложение письма, содержимое уже в памяти (см. FileService,
// который читает файл с диска перед вызовом Send).
type Attachment struct {
	Filename string
	MimeType string
	Data     []byte
}

// Service — общая отправка почты, не завязана на конкретный сценарий
// (утверждение отпуска и т.п. — забота вызывающей стороны, собирает
// тему/текст/вложения сама).
type Service interface {
	Send(ctx context.Context, to, subject, body string, attachments []Attachment) error
}

type service struct {
	settings smtpsettings.Service
}

// NewService — настройки SMTP (хост/порт/логин/пароль/from) читаются из
// system_settings при каждой отправке через settings, а не один раз при
// старте — изменения в "Настройках" применяются сразу, без рестарта.
func NewService(settings smtpsettings.Service) Service {
	return &service{settings: settings}
}

func (s *service) Send(ctx context.Context, to, subject, body string, attachments []Attachment) error {
	cfg, err := s.settings.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("get smtp config: %w", err)
	}
	if cfg.Host == "" {
		return errors.New("smtp is not configured")
	}
	if to == "" {
		return errors.New("recipient is required")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	for _, a := range attachments {
		data := a.Data
		m.Attach(a.Filename, gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := w.Write(data)
			return err
		}))
	}

	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	return d.DialAndSend(m)
}
