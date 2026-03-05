package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"rsps/internal/api"
	"rsps/internal/ast"
	"rsps/internal/db"
	"rsps/internal/migrations"
	"rsps/internal/parser"
	"rsps/internal/runtime"
	"rsps/internal/schema"
	"rsps/internal/ui"
	"rsps/internal/validator"
)

type BuildArtifacts struct {
	App      *ast.Application
	Schema   *schema.Schema
	Registry *runtime.Registry
	BuildDir string
}

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	appPath := os.Args[2]

	switch command {
	case "build":
		artifacts, err := build(appPath)
		if err != nil {
			log.Fatalf("build failed: %v", err)
		}
		log.Printf("build completed for app '%s'", artifacts.App.Name)
		log.Printf("metadata: %s", filepath.Join(artifacts.BuildDir, "metadata.json"))
		log.Printf("schema sql: %s", filepath.Join(artifacts.BuildDir, "schema.sql"))
	case "run":
		if err := run(appPath); err != nil {
			log.Fatalf("run failed: %v", err)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func build(appPath string) (*BuildArtifacts, error) {
	// Build pipeline: parse DSL -> validate semantics -> generate schema + metadata.
	parsedApp, err := parser.ParseFile(appPath)
	if err != nil {
		return nil, err
	}

	if err := validator.Validate(parsedApp); err != nil {
		return nil, err
	}

	generatedSchema, err := schema.Generate(parsedApp)
	if err != nil {
		return nil, err
	}

	registry := runtime.NewRegistry(parsedApp)
	buildDir := filepath.Join(filepath.Dir(appPath), ".rsps")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return nil, fmt.Errorf("create build directory '%s': %w", buildDir, err)
	}

	if err := os.WriteFile(filepath.Join(buildDir, "schema.sql"), []byte(generatedSchema.SQL()), 0644); err != nil {
		return nil, fmt.Errorf("write schema sql: %w", err)
	}

	serializedSchema, err := json.MarshalIndent(generatedSchema, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("serialize schema: %w", err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, "schema.json"), serializedSchema, 0644); err != nil {
		return nil, fmt.Errorf("write schema json: %w", err)
	}

	hash, err := generatedSchema.Hash()
	if err != nil {
		return nil, fmt.Errorf("hash schema: %w", err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, "schema.hash"), []byte(hash), 0644); err != nil {
		return nil, fmt.Errorf("write schema hash: %w", err)
	}

	if err := registry.Save(filepath.Join(buildDir, "metadata.json")); err != nil {
		return nil, err
	}

	return &BuildArtifacts{
		App:      parsedApp,
		Schema:   generatedSchema,
		Registry: registry,
		BuildDir: buildDir,
	}, nil
}

func run(appPath string) error {
	// Run reuses build output as single source of truth for migrations and runtime generation.
	artifacts, err := build(appPath)
	if err != nil {
		return err
	}

	databasePath := filepath.Join(filepath.Dir(appPath), artifacts.App.Name+".sqlite")
	connection, err := db.OpenSQLite(databasePath)
	if err != nil {
		return err
	}
	defer connection.Close()

	migrator := migrations.New(connection)
	if err := migrator.Apply(artifacts.Schema); err != nil {
		return err
	}

	repository := db.NewRepository(connection, artifacts.Registry)
	apiHandler := api.NewHandler(repository, artifacts.Registry)

	templateDir, err := resolveTemplateDir(appPath)
	if err != nil {
		return err
	}
	uiHandler, err := ui.NewHandler(repository, artifacts.Registry, templateDir)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	apiHandler.RegisterRoutes(mux)
	uiHandler.RegisterRoutes(mux)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", http.StatusFound)
	})

	address := os.Getenv("RSPS_ADDR")
	if address == "" {
		address = ":8080"
	}

	server := &http.Server{
		Addr:         address,
		Handler:      requestLogger(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	log.Printf("app '%s' running on http://localhost%s", artifacts.App.Name, address)
	log.Printf("database: %s", databasePath)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func resolveTemplateDir(appPath string) (string, error) {
	if explicit := os.Getenv("RSPS_TEMPLATE_DIR"); explicit != "" {
		if hasTemplates(explicit) {
			return explicit, nil
		}
		return "", fmt.Errorf("RSPS_TEMPLATE_DIR does not contain ui_list.html and ui_form.html: %s", explicit)
	}

	workingDirectory, _ := os.Getwd()
	executablePath, _ := os.Executable()

	candidates := []string{
		filepath.Join(workingDirectory, "templates"),
		filepath.Join(filepath.Dir(executablePath), "templates"),
		filepath.Join(filepath.Dir(appPath), "templates"),
		filepath.Join(filepath.Dir(filepath.Dir(appPath)), "templates"),
	}

	for _, candidate := range candidates {
		if hasTemplates(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("unable to locate templates directory")
}

func hasTemplates(path string) bool {
	listPath := filepath.Join(path, "ui_list.html")
	formPath := filepath.Join(path, "ui_form.html")
	if _, err := os.Stat(listPath); err != nil {
		return false
	}
	if _, err := os.Stat(formPath); err != nil {
		return false
	}
	return true
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  rsps build <path/to/app.rsps>")
	fmt.Fprintln(os.Stderr, "  rsps run <path/to/app.rsps>")
}
