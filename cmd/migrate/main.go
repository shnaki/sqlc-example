// Package main provides a migration tool.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var (
		path     string
		database string
		ext      string
		dir      string
		seq      bool
	)

	// Minimal flag support for migrate-up and migrate-down
	flag.StringVar(&path, "path", "", "path to migrations")
	flag.StringVar(&database, "database", "", "database connection string")

	// Support for 'create' command flags
	createCmd := flag.NewFlagSet("create", flag.ExitOnError)
	createCmd.StringVar(&ext, "ext", "sql", "extension")
	createCmd.StringVar(&dir, "dir", "", "directory")
	createCmd.BoolVar(&seq, "seq", true, "sequential versioning")

	if len(os.Args) < 2 {
		fmt.Println("usage: migrate [options] <command> [args]")
		os.Exit(1)
	}

	command := os.Args[1]

	flag.Parse()

	remaining := flag.Args()
	if len(remaining) > 0 {
		command = remaining[0]
	}

	switch command {
	case "up":
		if path == "" || database == "" {
			log.Fatal("-path and -database are required")
		}
		m, err := migrate.New("file://"+path, database)
		if err != nil {
			log.Fatal(err)
		}
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		fmt.Println("Migration up successful")

	case "down":
		if path == "" || database == "" {
			log.Fatal("-path and -database are required")
		}
		m, err := migrate.New("file://"+path, database)
		if err != nil {
			log.Fatal(err)
		}
		steps := 1
		if len(remaining) > 1 {
			if _, err := fmt.Sscanf(remaining[1], "%d", &steps); err != nil {
				log.Fatalf("invalid steps: %v", err)
			}
		}
		if err := m.Steps(-steps); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		fmt.Println("Migration down successful")

	case "create":
		if err := createCmd.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		args := createCmd.Args()
		var name string
		if len(args) == 0 {
			fmt.Print("Migration name: ")
			if _, err := fmt.Scanln(&name); err != nil {
				log.Fatal("migration name is required")
			}
		} else {
			name = args[0]
		}

		if dir == "" {
			dir = "."
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatal(err)
		}

		version := time.Now().Unix()
		if seq {
			entries, err := os.ReadDir(dir)
			if err != nil {
				log.Fatal(err)
			}
			maxVersion := 0
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				var v int
				_, err := fmt.Sscanf(entry.Name(), "%d", &v)
				if err != nil {
					continue
				}
				if v > maxVersion {
					maxVersion = v
				}
			}
			version = int64(maxVersion + 1)
		}

		for _, direction := range []string{"up", "down"} {
			filename := fmt.Sprintf("%06d_%s.%s.%s", version, name, direction, ext)
			path := dir + "/" + filename
			if _, err := os.Stat(path); err == nil {
				log.Fatalf("file already exists: %s", path)
			}
			err := os.WriteFile(path, []byte(""), 0644)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Created: %s\n", path)
		}

	case "version":
		if database == "" {
			log.Fatal("-database is required")
		}
		m, err := migrate.New("file://"+path, database)
		if err != nil {
			log.Fatal(err)
		}
		v, dirty, err := m.Version()
		if err != nil && err != migrate.ErrNilVersion {
			log.Fatal(err)
		}
		fmt.Printf("Version: %d, Dirty: %v\n", v, dirty)

	case "force":
		if database == "" {
			log.Fatal("-database is required")
		}
		if len(remaining) < 2 {
			log.Fatal("version number is required for force command")
		}
		var v int
		if _, err := fmt.Sscanf(remaining[1], "%d", &v); err != nil {
			log.Fatalf("invalid version: %v", err)
		}
		m, err := migrate.New("file://"+path, database)
		if err != nil {
			log.Fatal(err)
		}
		if err := m.Force(v); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Forced version to %d\n", v)

	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}
