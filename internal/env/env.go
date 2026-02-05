package env

import (
	"log"
	"os"

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

func (e *Env) GetSecretKey() string {
	return os.Getenv("SECRET_KEY")
}
