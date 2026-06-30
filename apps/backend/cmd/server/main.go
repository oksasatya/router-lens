package main

import (
	"flag"
	"log"

	"router-lens/internal/bootstrap"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "apply migrations then exit")
	flag.Parse()

	if *migrateOnly {
		if err := bootstrap.MigrateAndExit(); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		return
	}

	bootstrap.New().Run()
}
