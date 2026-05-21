package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pianke-ticket/backend/internal/models"
	"pianke-ticket/backend/internal/store"
)

func TestGenerateWebImage(t *testing.T) {
	png := base64.StdEncoding.EncodeToString([]byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	})
	var upstreamRequest map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("unexpected upstream path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&upstreamRequest); err != nil {
			t.Fatalf("decode upstream request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": []map[string]any{{
				"type":   "image_generation_call",
				"result": png,
			}},
		})
	}))
	defer upstream.Close()

	server := NewServer(ServerConfig{
		PublicBaseURL:     "https://berserk.test",
		XAIAPIKey:         "test-key",
		XAIBaseURL:        upstream.URL,
		GeneratedImageDir: t.TempDir(),
		Store:             &webImageTestStore{},
	})

	body := `{"prompt":"雨夜街头","style":"赛博朋克","n":1,"size":"1024x1365","quality":"medium","modelID":"gpt-image"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/generate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-session")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response models.WebImageGenerateResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Images) != 1 {
		t.Fatalf("expected one image, got %d", len(response.Images))
	}
	if !strings.HasPrefix(response.Images[0].URL, "https://berserk.test/generated/") {
		t.Fatalf("expected generated image URL, got %q", response.Images[0].URL)
	}
	if response.Credits != 5 || response.ModelID != "gpt-image" {
		t.Fatalf("unexpected generation metadata: %+v", response)
	}
	if upstreamRequest["model"] != "gpt-5.5" {
		t.Fatalf("unexpected upstream model: %v", upstreamRequest["model"])
	}
	tools, ok := upstreamRequest["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("expected one image tool, got %#v", upstreamRequest["tools"])
	}
	tool, ok := tools[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected image tool payload: %#v", tools[0])
	}
	if tool["size"] != "1024x1360" {
		t.Fatalf("expected normalized image size, got %v", tool["size"])
	}
}

func TestNormalizedImageSize(t *testing.T) {
	cases := map[string]string{
		"":          "1024x1360",
		"自动":        "1024x1360",
		"3:4":       "1024x1360",
		"4:3":       "1360x1024",
		"1024x1365": "1024x1360",
		"1365x1024": "1360x1024",
		"1024x1536": "1024x1536",
		"bad-size":  "1024x1360",
		"128x128":   "256x256",
		"5000x5000": "2048x2048",
	}
	for input, expected := range cases {
		if got := normalizedImageSize(input); got != expected {
			t.Fatalf("normalizedImageSize(%q) = %q, expected %q", input, got, expected)
		}
	}
}

func TestWebImageErrorMessageLocalizesInvalidSize(t *testing.T) {
	message := webImageErrorMessage(errors.New(`{"error":{"message":"Invalid size '1024x1365'. Width and height must both be divisible by 16.","code":"invalid_value"}}`))
	if !strings.Contains(message, "图片尺寸不符合模型要求") {
		t.Fatalf("expected localized size error, got %q", message)
	}
}

type webImageTestStore struct{}

func (s *webImageTestStore) CreateEmailUser(context.Context, string, string, string) (models.User, error) {
	return models.User{}, nil
}
func (s *webImageTestStore) GetEmailUser(context.Context, string, string) (models.User, error) {
	return models.User{}, store.ErrNotFound
}
func (s *webImageTestStore) GetEmailPasswordHash(context.Context, string, string) (string, error) {
	return "", store.ErrNotFound
}
func (s *webImageTestStore) SetEmailUserPassword(context.Context, string, string, string) error {
	return nil
}
func (s *webImageTestStore) SaveEmailCode(context.Context, string, string, string, string, time.Time) error {
	return nil
}
func (s *webImageTestStore) VerifyEmailCode(context.Context, string, string, string, string, string, time.Time) error {
	return nil
}
func (s *webImageTestStore) ConsumeVerifiedEmailCode(context.Context, string, string, string, string) error {
	return nil
}
func (s *webImageTestStore) ConsumeEmailCode(context.Context, string, string, string, string) error {
	return nil
}
func (s *webImageTestStore) CreateSession(context.Context, string, time.Time) (string, error) {
	return "test-session", nil
}
func (s *webImageTestStore) GetUserBySession(context.Context, string) (models.User, error) {
	return models.User{ID: "00000000-0000-0000-0000-000000000001", AppID: "berserk.web", Email: "qa@example.com", Credits: 95}, nil
}
func (s *webImageTestStore) GetUser(context.Context, string) (models.User, error) {
	return models.User{ID: "00000000-0000-0000-0000-000000000001", AppID: "berserk.web", Email: "qa@example.com", Credits: 95}, nil
}
func (s *webImageTestStore) UpdateUserProfile(context.Context, string, models.UserProfileUpdateRequest) (models.User, error) {
	return models.User{}, nil
}
func (s *webImageTestStore) DeleteUser(context.Context, string) error { return nil }
func (s *webImageTestStore) AddCredits(context.Context, string, int, string, string, string) (int, error) {
	return 100, nil
}
func (s *webImageTestStore) ConsumeCredits(context.Context, string, int, string, string, string) (int, error) {
	return 95, nil
}
func (s *webImageTestStore) CreateCreditOrder(context.Context, string, models.CreditPackage) (models.CreditOrder, error) {
	return models.CreditOrder{}, nil
}
func (s *webImageTestStore) ListCreditPackages(context.Context) ([]models.CreditPackage, error) {
	return nil, nil
}
func (s *webImageTestStore) RedeemCreditCode(context.Context, string, string, string) (int, error) {
	return 0, store.ErrNotFound
}
func (s *webImageTestStore) ListImageModels(context.Context) ([]models.ImageModel, error) {
	return []models.ImageModel{{ID: "gpt-image", Name: "GPT Image", CreditCost: 5, Enabled: true}}, nil
}
func (s *webImageTestStore) GetImageModel(context.Context, string) (models.ImageModel, error) {
	return models.ImageModel{ID: "gpt-image", Name: "GPT Image", CreditCost: 5, Enabled: true}, nil
}
func (s *webImageTestStore) CreateGalleryImages(_ context.Context, userID string, prompt string, style string, modelID string, modelName string, size string, quality string, creditsCost int, images []models.WebGeneratedImage) ([]models.WebGalleryImage, error) {
	items := make([]models.WebGalleryImage, 0, len(images))
	for _, image := range images {
		items = append(items, models.WebGalleryImage{ID: "generated", UserID: userID, Image: image.URL, Prompt: prompt, Style: style, ModelID: modelID, ModelName: modelName, Size: size, Quality: quality, CreditsCost: creditsCost, IsPublic: true, CreatedAt: time.Now().UTC().Format(time.RFC3339)})
	}
	return items, nil
}
func (s *webImageTestStore) ListGalleryImages(context.Context, string, int, string, string) ([]models.WebGalleryImage, error) {
	return nil, nil
}
func (s *webImageTestStore) ListFavoriteGalleryImages(context.Context, string, int, string, string) ([]models.WebGalleryImage, error) {
	return nil, nil
}
func (s *webImageTestStore) SetGalleryImageLike(context.Context, string, string, bool) (models.WebGalleryImage, error) {
	return models.WebGalleryImage{}, store.ErrNotFound
}
func (s *webImageTestStore) SetGalleryImageFavorite(context.Context, string, string, bool) (models.WebGalleryImage, error) {
	return models.WebGalleryImage{}, store.ErrNotFound
}
func (s *webImageTestStore) SetGalleryImageFeatured(context.Context, string, string, bool, bool) (models.WebGalleryImage, error) {
	return models.WebGalleryImage{}, store.ErrNotFound
}
func (s *webImageTestStore) HasActiveWebImageTask(context.Context, string) (bool, error) {
	return false, nil
}
func (s *webImageTestStore) CreateWebImageTask(context.Context, string, string, string, string, string, string, int, int) (models.WebImageTask, error) {
	return models.WebImageTask{}, nil
}
func (s *webImageTestStore) ListWebImageTasks(context.Context, string, int) ([]models.WebImageTask, error) {
	return nil, nil
}
func (s *webImageTestStore) GetWebImageTask(context.Context, string, string) (models.WebImageTask, error) {
	return models.WebImageTask{}, store.ErrNotFound
}
func (s *webImageTestStore) MarkWebImageTaskRunning(context.Context, string, string) error {
	return nil
}
func (s *webImageTestStore) CompleteWebImageTask(context.Context, string, string, models.WebGeneratedImage, string) (models.WebImageTask, error) {
	return models.WebImageTask{}, nil
}
func (s *webImageTestStore) FailWebImageTask(context.Context, string, string, string) (models.WebImageTask, error) {
	return models.WebImageTask{}, nil
}
func (s *webImageTestStore) SetWebImageTaskPublic(context.Context, string, string, bool) (models.WebImageTask, error) {
	return models.WebImageTask{}, nil
}
