package env

import (
	"os"

	"github.com/joho/godotenv"
)

type Environment struct {
	HOMEASSISTANT_BASE_URL string
	HOMEASSISTANT_TOKEN    string
}

func Load() Environment {
	godotenv.Load()       // project root (normal run)
	godotenv.Load("../.env") // subpackage tests
	return Environment{
		HOMEASSISTANT_BASE_URL: os.Getenv("HOMEASSISTANT_BASE_URL"),
		HOMEASSISTANT_TOKEN:    os.Getenv("HOMEASSISTANT_TOKEN"),
	}
}
