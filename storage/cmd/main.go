package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"storage/internal/handlers"
	"storage/internal/repository"
	"storage/internal/service"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	minioClient := initMinIO()
	bucket := os.Getenv("MINIO_BUCKET")
	if bucket == "" {
		bucket = "chunks"
	}

	repository := repository.NewRepository(minioClient, bucket)
	storageService := service.NewStorageService(repository)
	storageHandler := handlers.NewStorageHandler(storageService)

	muxRouter := mux.NewRouter()
	muxRouter.Use(corsMiddleware)

	muxRouter.HandleFunc("/api/chunks/upload", storageHandler.UploadChunk).Methods("POST")
	muxRouter.HandleFunc("/api/chunks/download", storageHandler.DownloadChunk).Methods("GET")

	muxRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}).Methods("GET")

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8081"
	}

	if err := http.ListenAndServe(":"+port, muxRouter); err != nil {
		log.Fatal("Error starting HTTP server:", err)
	}
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

func initMinIO() *minio.Client {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "minio:9000"
	}

	accessKeyID := os.Getenv("MINIO_ACCESS_KEY_ID")
	if accessKeyID == "" {
		accessKeyID = "minioadmin"
	}

	secretAccessKey := os.Getenv("MINIO_SECRET_ACCESS_KEY")
	if secretAccessKey == "" {
		secretAccessKey = "minioadmin"
	}

	useSSL := false

	var client *minio.Client
	var err error

	maxRetries := 10
	for i := range maxRetries {
		client, err = minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
			Secure: useSSL,
		})
		if err != nil {
			log.Printf("Error creating MinIO client (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(2 * time.Second)
			continue
		}

		bucketName := os.Getenv("MINIO_BUCKET")
		if bucketName == "" {
			bucketName = "chunks"
		}

		exists, err := client.BucketExists(context.Background(), bucketName)
		if err != nil {
			log.Printf("Error checking bucket existence (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(2 * time.Second)
			continue
		}

		if !exists {
			err = client.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
			if err != nil {
				log.Printf("Error creating bucket (attempt %d/%d): %v", i+1, maxRetries, err)
				time.Sleep(2 * time.Second)
				continue
			}
		}

		return client
	}

	log.Fatal("Could not connect to MinIO after retries:", err)
	return nil
}
