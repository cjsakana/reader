package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type Config struct {
	Server   Server
	Database Database
	Upload   Upload
	LLM      LLMConfig    `yaml:"llm"`
	Milvus   MilvusConfig `yaml:"milvus"`
}

type Server struct {
	Port string
}
type Database struct {
	DBPath string
}
type Upload struct {
	BooksDir string
}

// LLMConfig holds LLM service settings.
type LLMConfig struct {
	BaseURL    string `yaml:"base_url"`
	APIKey     string `yaml:"api_key"`
	Model      string `yaml:"model"`
	EmbedModel string `yaml:"embed_model"`
}

// MilvusConfig holds Milvus vector database settings.
type MilvusConfig struct {
	Address    string `yaml:"address"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	DBName     string `yaml:"db_name"`
	Collection string `yaml:"collection"`
	Dimension  int    `yaml:"dimension"`
}

func Load(path string) *Config {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		// File not found — fall back to defaults silently.
		return cfg
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		// Malformed YAML — fall back to defaults.
		fmt.Println(err)
		return cfg
	}

	// Ensure directories exist
	os.MkdirAll(cfg.Upload.BooksDir, 0755)

	return cfg
}

func DefaultConfig() *Config {
	return &Config{
		Server:   Server{Port: "8080"},
		Database: Database{DBPath: filepath.Join(".", "data", "reader.db")},
		Upload:   Upload{BooksDir: filepath.Join(".", "data", "books")},
		LLM: LLMConfig{
			BaseURL:    "http://localhost:11434/v1",
			APIKey:     "sk-placeholder",
			Model:      "gpt-4o-mini",
			EmbedModel: "text-embedding-3-small",
		},
		Milvus: MilvusConfig{
			Address:    "localhost:19530",
			Username:   "",
			Password:   "",
			DBName:     "reader",
			Collection: "novel_chunks",
			Dimension:  1536,
		},
	}
}
