package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type appConfig struct {
	variant     string
	httpAddr    string
	mysqlDSN    string
	postgresDSN string
	mongoURI    string
}

type userResponse struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Tier   string `json:"tier"`
	Source string `json:"source"`
}

func main() {
	cfg := appConfig{
		variant:     env("API_VARIANT", "old"),
		httpAddr:    env("HTTP_ADDR", ":8080"),
		mysqlDSN:    os.Getenv("MYSQL_DSN"),
		postgresDSN: os.Getenv("POSTGRES_DSN"),
		mongoURI:    os.Getenv("MONGO_URI"),
	}

	if err := cfg.validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/users/1", cfg.handleUser)

	log.Printf("starting %s demo API on %s", cfg.variant, cfg.httpAddr)
	if err := http.ListenAndServe(cfg.httpAddr, mux); err != nil {
		log.Fatal(err)
	}
}

func (cfg appConfig) validate() error {
	switch cfg.variant {
	case "old", "new":
	default:
		return fmt.Errorf("API_VARIANT must be old or new, got %q", cfg.variant)
	}
	if cfg.mysqlDSN == "" {
		return errors.New("MYSQL_DSN is required")
	}
	if cfg.postgresDSN == "" {
		return errors.New("POSTGRES_DSN is required")
	}
	if cfg.mongoURI == "" {
		return errors.New("MONGO_URI is required")
	}
	return nil
}

func (cfg appConfig) handleUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	name, err := cfg.queryMySQL(ctx)
	if err != nil {
		http.Error(w, "mysql query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tier, err := cfg.queryPostgres(ctx)
	if err != nil {
		http.Error(w, "postgres query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := cfg.queryMongo(ctx); err != nil {
		http.Error(w, "mongo query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(userResponse{
		ID:     1,
		Name:   name,
		Tier:   tier,
		Source: "shadiff-e2e",
	})
}

func (cfg appConfig) queryMySQL(ctx context.Context) (string, error) {
	db, err := sql.Open("mysql", cfg.mysqlDSN)
	if err != nil {
		return "", err
	}
	defer db.Close()

	query := "SELECT name FROM users WHERE id = 1 /* old_mysql_lookup */"
	if cfg.variant == "new" {
		query = "SELECT name FROM users WHERE id = 1 /* new_mysql_lookup */"
	}

	var name string
	if err := db.QueryRowContext(ctx, query).Scan(&name); err != nil {
		return "", err
	}
	return name, nil
}

func (cfg appConfig) queryPostgres(ctx context.Context) (string, error) {
	db, err := sql.Open("postgres", cfg.postgresDSN)
	if err != nil {
		return "", err
	}
	defer db.Close()

	query := "SELECT tier FROM accounts WHERE id = 1 /* old_postgres_lookup */"
	if cfg.variant == "new" {
		query = "SELECT tier FROM accounts WHERE id = 1 /* new_postgres_lookup */"
	}

	var tier string
	if err := db.QueryRowContext(ctx, query).Scan(&tier); err != nil {
		return "", err
	}
	return tier, nil
}

func (cfg appConfig) queryMongo(ctx context.Context) error {
	client, err := mongo.Connect(options.Client().ApplyURI(cfg.mongoURI))
	if err != nil {
		return err
	}
	defer client.Disconnect(context.Background())

	collection := "users_old"
	if cfg.variant == "new" {
		collection = "users_new"
	}

	var doc struct {
		ID   int    `bson:"id"`
		Name string `bson:"name"`
	}
	return client.Database("shadiff").Collection(collection).FindOne(ctx, bson.D{{Key: "id", Value: 1}}).Decode(&doc)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
