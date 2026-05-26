package config

import (
	"flag"
	"log"
	"path/filepath"
	"strings"
)

const (
	dataFileName       = "sites.db"
	legacyJSONFileName = "sites.json"
	defaultDataDir     = "data"
	defaultPort        = 8080
)

// Config 保存服务启动时解析出来的运行配置。
type Config struct {
	Port           int
	DataDir        string
	DataPath       string
	LegacyJSONPath string
}

// ParseFlags 解析命令行参数，并生成 SQLite 与旧 JSON 数据文件路径。
func ParseFlags() Config {
	port := flag.Int("port", defaultPort, "HTTP server port")
	dataDir := flag.String("data", defaultDataDir, "directory for SQLite data file")
	flag.Parse()

	if *port < 1 || *port > 65535 {
		log.Fatalf("端口必须在 1 到 65535 之间: %d", *port)
	}

	trimmedDataDir := strings.TrimSpace(*dataDir)
	cleanDataDir := filepath.Clean(trimmedDataDir)
	if cleanDataDir == "." && trimmedDataDir == "" {
		log.Fatal("数据目录不能为空")
	}

	return Config{
		Port:           *port,
		DataDir:        cleanDataDir,
		DataPath:       filepath.Join(cleanDataDir, dataFileName),
		LegacyJSONPath: filepath.Join(cleanDataDir, legacyJSONFileName),
	}
}
