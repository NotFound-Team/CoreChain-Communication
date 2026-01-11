package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("UploadHandler called")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(50 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Println("Error getting file from request:", err)
		http.Error(w, "File not found", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	fileID := uuid.New().String()
	objectName := fmt.Sprintf("%s%s", fileID, ext)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	info, err := Instance.Client.PutObject(ctx, Instance.Bucket, objectName, file, header.Size, minio.PutObjectOptions{
		ContentType: header.Header.Get("Content-Type"),
	})
	if err != nil {
		log.Println("Error uploading file to MinIO:", err)
		http.Error(w, "Upload failed", http.StatusInternalServerError)
		return
	}

	fileData := map[string]interface{}{
		"file_id":   info.Key,
		"file_name": header.Filename,
		"file_path": info.Key,
		"file_type": header.Header.Get("Content-Type"),
		"file_size": header.Size,
	}

	response := map[string]interface{}{
		"success": true,
		"message": "File uploaded successfully",
		"data":    fileData,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding upload response: %v", err)
	}
}
