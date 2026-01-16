package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Joseda-hg/lazytask/internal/config"
	"github.com/Joseda-hg/lazytask/internal/db"
	"github.com/Joseda-hg/lazytask/internal/tui"
	"github.com/Joseda-hg/lazytask/internal/web"
)

func main() {
	configPathFlag := flag.String("config", "", "config file path")
	dbPathFlag := flag.String("db", "", "sqlite db path")
	webFlag := flag.Bool("web", false, "enable web server")
	webOnlyFlag := flag.Bool("web-only", false, "run web server only")
	portFlag := flag.Int("port", 0, "web server port")
	flag.Parse()

	cfgPath, err := resolveConfigPath(*configPathFlag)
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal(err)
	}

	if *dbPathFlag != "" {
		cfg.DBPath = *dbPathFlag
	}
	if cfg.DBPath == "" {
		cfg.DBPath = filepath.Join(filepath.Dir(cfgPath), "lazytask.db")
	}
	if *webFlag {
		cfg.WebEnabled = true
	}
	if *portFlag != 0 {
		cfg.WebPort = *portFlag
	}
	if cfg.WebPort == 0 {
		cfg.WebPort = 8080
	}

	if err := config.Save(cfgPath, cfg); err != nil {
		log.Fatal(err)
	}

	store, err := openStore(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}

	if cfg.WebEnabled {
		addr := fmt.Sprintf(":%d", cfg.WebPort)
		handler := web.NewServer(store).Handler()
		if *webOnlyFlag {
			log.Printf("Web server running at http://localhost%s", addr)
			log.Fatal(http.ListenAndServe(addr, handler))
		}

		go func() {
			log.Printf("Web server running at http://localhost%s", addr)
			if err := http.ListenAndServe(addr, handler); err != nil {
				log.Printf("web server error: %v", err)
			}
		}()
	}

	if *webOnlyFlag {
		return
	}

	if err := tui.Run(store); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func resolveConfigPath(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	return config.DefaultConfigPath()
}

func openStore(dbPath string) (*db.Store, error) {
	if err := config.EnsureDir(dbPath); err != nil {
		return nil, err
	}

	sqlDB, err := db.Open(dbPath)
	if err != nil {
		return nil, err
	}

	return db.NewStore(sqlDB), nil
}
