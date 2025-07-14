package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gateway/internal/handlers"
	"gateway/internal/repository"
	"gateway/internal/service"
	"gateway/internal/storage"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	db := initDB()
	repository := repository.NewRepository(db)

	storageAddrs := getStorageAddresses()

	storageManager, err := storage.NewStorageManager(storageAddrs)
	if err != nil {
		log.Fatal("Error creating storage manager:", err)
	}
	defer storageManager.Close()

	log.Printf("Initialized storage manager with %d storage instances", storageManager.GetNumStorage())

	chunkerService := service.NewChunkerService(repository, storageManager)
	gatewayHandler := handlers.NewGatewayHandler(chunkerService)

	muxRouter := mux.NewRouter()
	muxRouter.Use(corsMiddleware)

	muxRouter.HandleFunc("/api/files/upload", gatewayHandler.UploadFile).Methods("POST")
	muxRouter.HandleFunc("/api/files/get", gatewayHandler.GetFile).Methods("GET")

	muxRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := http.ListenAndServe(":"+port, muxRouter); err != nil {
		log.Fatal("Error starting server:", err)
	}
}

func getStorageAddresses() []string {
	if addrs := os.Getenv("STORAGE_ADDRESSES"); addrs != "" {
		return strings.Split(addrs, ",")
	}

	if addr := os.Getenv("STORAGE_HTTP_ADDR"); addr != "" {
		return []string{addr}
	}

	var addrs []string
	numStorage := 10
	if envNum := os.Getenv("NUM_STORAGE_INSTANCES"); envNum != "" {
		if n, err := strconv.Atoi(envNum); err == nil && n > 0 && n <= 20 {
			numStorage = n
		}
	}

	for i := 1; i <= numStorage; i++ {
		addrs = append(addrs, fmt.Sprintf("storage-%d:8081", i))
	}

	return addrs
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func initDB() *sqlx.DB {
	dsn := os.Getenv("DSN")
	var db *sqlx.DB
	var err error

	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		db, err = sqlx.Open("pgx", dsn)
		if err != nil {
			log.Printf("Error opening database (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(2 * time.Second)
			continue
		}

		pingErr := db.Ping()
		if pingErr == nil {
			return db
		}

		log.Printf("Database ping failed (attempt %d/%d): %v", i+1, maxRetries, pingErr)
		db.Close()
		time.Sleep(2 * time.Second)
	}

	log.Fatal("Could not connect to database after retries:", err)
	return nil
}
