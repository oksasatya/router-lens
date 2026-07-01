package main

import (
	"flag"
	"log"

	"router-lens/internal/platform/bootstrap"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "apply migrations then exit")
	createAdmin := flag.Bool("create-admin", false, "create the first admin user then exit")
	email := flag.String("email", "", "admin email (required with -create-admin)")
	password := flag.String("password", "", "admin password (required with -create-admin)")
	name := flag.String("name", "", "admin display name (optional)")
	flag.Parse()

	if *migrateOnly {
		if err := bootstrap.MigrateAndExit(); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		return
	}
	if *createAdmin {
		if *email == "" || *password == "" {
			log.Fatal("create-admin: -email and -password are required")
		}
		if err := bootstrap.CreateAdminAndExit(*email, *password, *name); err != nil {
			log.Fatalf("create-admin: %v", err)
		}
		return
	}

	bootstrap.New().Run()
}
