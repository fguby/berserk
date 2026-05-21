package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"pianke-ticket/backend/internal/models"
)

func (s *Server) callAinaibaImage(ctx context.Context, request models.MangaGenerateRequest) ([]string, error) {
	if strings.TrimSpace(s.xaiAPIKey) == "" {
		return nil, errors.New("xai_api_key is not configured")
	}
	n := request.N
	if n < 1 {
		n = 1
	}
	if n > 4 {
		n = 4
	}
	tool := map[string]any{
		"type":          "image_generation",
		"model":         firstNonEmpty(s.xaiImageModel, "gpt-image-2"),
		"size":          normalizedImageSize(request.Size),
		"quality":       firstNonEmpty(request.Quality, "medium"),
		"output_format": "png",
	}
	input := []map[string]any{{
		"role":    "user",
		"content": buildResponsesContent(request.Prompt, request.Images),
	}}
	var images []string
	for i := 0; i < n; i++ {
		data, err := s.postAinaiba(ctx, map[string]any{
			"model": firstNonEmpty(s.xaiMainModel, "gpt-5.5"),
			"input": input,
			"tools": []map[string]any{tool},
		})
		if err != nil {
			return nil, err
		}
		images = append(images, extractImageResults(data)...)
	}
	if len(images) == 0 {
		return nil, errors.New("no image result returned")
	}
	return images, nil
}

func (s *Server) postAinaiba(ctx context.Context, body map[string]any) (map[string]any, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	url := s.xaiBaseURL + s.xaiResponsesPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(s.xaiAPIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respData, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respData))
		if msg == "" {
			msg = resp.Status
		}
		return nil, errors.New(msg)
	}
	var decoded map[string]any
	if err := json.Unmarshal(respData, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func buildResponsesContent(prompt string, images []string) []map[string]any {
	content := []map[string]any{{"type": "input_text", "text": prompt}}
	for _, image := range images {
		image = strings.TrimSpace(image)
		if image != "" {
			content = append(content, map[string]any{"type": "input_image", "image_url": image})
		}
	}
	return content
}

func extractImageResults(root any) []string {
	var results []string
	var walk func(any)
	walk = func(value any) {
		switch v := value.(type) {
		case map[string]any:
			if v["type"] == "image_generation_call" {
				if result, ok := v["result"].(string); ok && result != "" {
					results = append(results, result)
				}
			}
			for _, child := range v {
				walk(child)
			}
		case []any:
			for _, child := range v {
				walk(child)
			}
		}
	}
	walk(root)
	return results
}
