package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	vision "cloud.google.com/go/vision/apiv1"
	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"
)

type Label struct {
	Description string  `json:"description"`
	Score       float32 `json:"score"`
}

type ApiResponse struct {
	Labels []Label `json:"labels,omitempty"`
	Error  string  `json:"error,omitempty"`
}

func main() {
	ctx := context.Background()
	// Vision API 클라이언트 생성 :contentReference[oaicite:0]{index=0}
	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		log.Fatalf("failed to create vision client: %v", err)
	}
	defer client.Close()

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./frontend"))
	mux.Handle("/", fs)

	// 이미지 업로드 API
	mux.HandleFunc("/api/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 10MB 제한
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

		file, _, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "이미지 파일을 읽을 수 없습니다: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// 업로드된 파일에서 Image 객체 생성 :contentReference[oaicite:1]{index=1}
		img, err := vision.NewImageFromReader(file)
		if err != nil {
			http.Error(w, "이미지 파싱 실패: "+err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Vision API에 LABEL_DETECTION 요청 :contentReference[oaicite:2]{index=2}
		res, err := client.AnnotateImage(ctx, &pb.AnnotateImageRequest{
			Image: img,
			Features: []*pb.Feature{
				{Type: pb.Feature_LABEL_DETECTION, MaxResults: 10},
			},
		})
		if err != nil {
			log.Printf("vision api error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(ApiResponse{Error: "Vision API 호출 실패"})
			return
		}

		labels := make([]Label, 0, len(res.LabelAnnotations))
		for _, l := range res.LabelAnnotations {
			labels = append(labels, Label{
				Description: l.GetDescription(),
				Score:       l.GetScore(),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ApiResponse{Labels: labels})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}