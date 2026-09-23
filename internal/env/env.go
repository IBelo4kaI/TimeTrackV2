package env

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Env struct {
}

func (e *Env) Init() error {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	return nil
}

func (e *Env) GetDbString() string {
	return os.Getenv("DB_STRING")
}

func (e *Env) GetAddr() string {
	return os.Getenv("ADDR")
}

func (e *Env) GetGRPCServerAddr() string {
	return os.Getenv("GRPC_ADDR")
}

// GetFrontendURL — публичный адрес фронта (без слэша на конце), нужен,
// чтобы собрать кликабельную ссылку на чат в VK-уведомлении.
func (e *Env) GetFrontendURL() string {
	return os.Getenv("FRONTEND_URL")
}

// GetVKGroupID — числовой id сообщества (ожидает vksdk/callback), не токен.
func (e *Env) GetVKGroupID() int {
	id, _ := strconv.Atoi(os.Getenv("VK_GROUP_ID"))
	return id
}

func (e *Env) GetVKGroupToken() string {
	return os.Getenv("VK_GROUP_TOKEN")
}

func (e *Env) GetVKConfirmationString() string {
	return os.Getenv("VK_CONFIRMATION_STRING")
}

func (e *Env) GetVKSecretKey() string {
	return os.Getenv("VK_SECRET_KEY")
}

// GetVKCommunityScreenName — короткое имя сообщества (vk.ru/<имя>), для
// диплинка привязки VK (vk.me/<имя>?ref=<код>) — см. vk.Service.GenerateLinkCode.
func (e *Env) GetVKCommunityScreenName() string {
	return os.Getenv("VK_COMMUNITY_SCREEN_NAME")
}

// GetAuthServiceHost — базовый адрес сервиса авторизации (без завершающего
// слэша), напр. "http://localhost:8382" — см. internal/authservice.
func (e *Env) GetAuthServiceHost() string {
	return os.Getenv("AUTH_SERVICE_HOST")
}

// GetAuthServiceAPIKey — отправляется как значение cookie "session" в
// запросах к сервису авторизации (см. authservice.Service.GetAllUsers).
func (e *Env) GetAuthServiceAPIKey() string {
	return os.Getenv("AUTH_SERVICE_API_KEY")
}

// GetSMTPEncryptionKey — ключ для шифрования пароля SMTP в system_settings
// (сам хост/порт/логин/пароль настраиваются в "Настройках", не в env — см.
// internal/smtp_settings). Пароль хранится AES-GCM, ключ шифрования — sha256
// от этой строки, так что подойдёт любой секрет любой длины.
func (e *Env) GetSMTPEncryptionKey() string {
	return os.Getenv("SMTP_ENCRYPTION_KEY")
}
