package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"navigation/internal/config"
	"navigation/internal/service"
	"navigation/internal/storage"
	httptransport "navigation/internal/transport/http"
)

//go:embed web/dist/*
var staticFiles embed.FS

func main() {
	cfg := config.ParseFlags()

	store, err := storage.NewSQLiteSiteStore(cfg.DataPath, cfg.LegacyJSONPath)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer store.Close()

	svc := service.NewSiteService(store)
	authSvc, err := service.NewAuthService(store)
	if err != nil {
		log.Fatalf("初始化账号失败: %v", err)
	}
	if cfg.ResetAuth {
		if err := authSvc.ResetDefaultUser(); err != nil {
			log.Fatalf("重置账号失败: %v", err)
		}
		log.Printf("账号密码已重置为: %s/%s", service.DefaultUsername, service.DefaultPassword)
		return
	}
	dist, err := fs.Sub(staticFiles, "web/dist")
	if err != nil {
		log.Fatalf("加载前端资源失败: %v", err)
	}
	handler := httptransport.NewHandler(svc, authSvc, dist)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("导航站已启动: http://localhost%s", addr)
	log.Printf("SQLite 数据库: %s", cfg.DataPath)
	if err := http.ListenAndServe(addr, handler.Routes()); err != nil {
		log.Fatal(err)
	}
}
