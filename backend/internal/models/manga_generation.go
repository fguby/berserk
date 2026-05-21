package models

type MangaGenerateRequest struct {
	Prompt  string   `json:"prompt"`
	Images  []string `json:"images,omitempty"`
	N       int      `json:"n,omitempty"`
	Size    string   `json:"size,omitempty"`
	Quality string   `json:"quality,omitempty"`
}

type WebImageGenerateRequest struct {
	Prompt         string   `json:"prompt"`
	Style          string   `json:"style,omitempty"`
	ModelID        string   `json:"modelID,omitempty"`
	Images         []string `json:"images,omitempty"`
	N              int      `json:"n,omitempty"`
	Size           string   `json:"size,omitempty"`
	Quality        string   `json:"quality,omitempty"`
	NegativePrompt string   `json:"negativePrompt,omitempty"`
	Resolution     string   `json:"resolution,omitempty"`
	LockedSeed     bool     `json:"lockedSeed,omitempty"`
}

type WebGeneratedImage struct {
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnailURL,omitempty"`
	MimeType     string `json:"mimeType"`
}

type ImageModel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Provider    string `json:"provider,omitempty"`
	Description string `json:"description,omitempty"`
	CreditCost  int    `json:"creditCost"`
	Enabled     bool   `json:"enabled"`
}

type ImageModelsResponse struct {
	Items []ImageModel `json:"items"`
}

type WebGalleryImage struct {
	ID               string `json:"id"`
	UserID           string `json:"userID,omitempty"`
	Author           string `json:"author,omitempty"`
	AuthorAvatarURL  string `json:"authorAvatarURL,omitempty"`
	Image            string `json:"image"`
	ThumbnailURL     string `json:"thumbnailURL,omitempty"`
	Prompt           string `json:"prompt"`
	Style            string `json:"style,omitempty"`
	ModelID          string `json:"modelID,omitempty"`
	ModelName        string `json:"modelName,omitempty"`
	Tag              string `json:"tag,omitempty"`
	Ratio            string `json:"ratio"`
	Size             string `json:"size,omitempty"`
	Quality          string `json:"quality,omitempty"`
	CreditsCost      int    `json:"creditsCost,omitempty"`
	IsPublic         bool   `json:"isPublic"`
	IsFeatured       bool   `json:"isFeatured"`
	IsPromptFeatured bool   `json:"isPromptFeatured"`
	LikeCount        int    `json:"likeCount"`
	LikedByMe        bool   `json:"likedByMe"`
	FavoriteCount    int    `json:"favoriteCount"`
	FavoritedByMe    bool   `json:"favoritedByMe"`
	CreatedAt        string `json:"createdAt"`
}

type WebGalleryResponse struct {
	Items []WebGalleryImage `json:"items"`
}

type WebImageGenerateResponse struct {
	Images    []WebGeneratedImage `json:"images"`
	Prompt    string              `json:"prompt"`
	Style     string              `json:"style,omitempty"`
	ModelID   string              `json:"modelID,omitempty"`
	ModelName string              `json:"modelName,omitempty"`
	Size      string              `json:"size"`
	Quality   string              `json:"quality"`
	Credits   int                 `json:"credits"`
	User      *User               `json:"user,omitempty"`
	CreatedAt string              `json:"createdAt"`
}

type WebImageTask struct {
	ID             string `json:"id"`
	UserID         string `json:"userID,omitempty"`
	Prompt         string `json:"prompt"`
	Style          string `json:"style,omitempty"`
	ModelID        string `json:"modelID,omitempty"`
	ModelName      string `json:"modelName,omitempty"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	N              int    `json:"n"`
	CreditsCost    int    `json:"creditsCost"`
	Status         string `json:"status"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
	ResultImage    string `json:"resultImage,omitempty"`
	ResultMimeType string `json:"resultMimeType,omitempty"`
	GalleryImageID string `json:"galleryImageID,omitempty"`
	IsPublic       bool   `json:"isPublic"`
	CreatedAt      string `json:"createdAt"`
	StartedAt      string `json:"startedAt,omitempty"`
	CompletedAt    string `json:"completedAt,omitempty"`
	UpdatedAt      string `json:"updatedAt,omitempty"`
}

type WebImageTaskResponse struct {
	Task WebImageTask `json:"task"`
	User *User        `json:"user,omitempty"`
}

type WebImageTasksResponse struct {
	Items []WebImageTask `json:"items"`
}

type WebImageTaskVisibilityRequest struct {
	IsPublic bool `json:"isPublic"`
}

type WebGalleryLikeRequest struct {
	Liked bool `json:"liked"`
}

type WebGalleryLikeResponse struct {
	Item WebGalleryImage `json:"item"`
}

type WebGalleryFavoriteRequest struct {
	Favorited bool `json:"favorited"`
}

type WebGalleryFavoriteResponse struct {
	Item WebGalleryImage `json:"item"`
}

type WebGalleryFeaturedRequest struct {
	IsFeatured       bool `json:"isFeatured"`
	IsPromptFeatured bool `json:"isPromptFeatured"`
}

type WebGalleryFeaturedResponse struct {
	Item WebGalleryImage `json:"item"`
}
