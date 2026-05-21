package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pianke-ticket/backend/internal/models"
	"pianke-ticket/backend/internal/store"

	"github.com/labstack/echo/v4"
)

func (s *Server) generateWebImage(c echo.Context) error {
	user, ok := s.requireUser(c)
	if !ok {
		return nil
	}
	var request models.WebImageGenerateRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "生图请求参数不正确"})
	}

	prompt := strings.TrimSpace(request.Prompt)
	if prompt == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "请输入提示词"})
	}

	size := normalizedImageSize(request.Size)
	request.Size = size
	request.Style = strings.TrimSpace(request.Style)
	quality := normalizedImageQuality(request.Quality)
	request.Quality = quality
	generationCount := normalizedGenerationCount(request.N)
	imageModel, err := s.resolveImageModel(c.Request().Context(), request.ModelID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "请选择可用的生图模型"})
	}
	creditsCost := generationCount * imageModel.CreditCost
	if _, err := s.store.ConsumeCredits(c.Request().Context(), user.ID, creditsCost, "web_image_generation", "web_image", ""); errors.Is(err, store.ErrInsufficientCredits) {
		return c.JSON(http.StatusPaymentRequired, models.ErrorResponse{Message: "积分不足，请先充值"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "扣减积分失败，请稍后重试"})
	}

	fullPrompt := buildWebImagePrompt(prompt, request.Style, request)
	s.logWebImagePrompt("web image prompt", imageModel.ID, request.Style, size, fullPrompt)
	images, err := s.callAinaibaImage(c.Request().Context(), models.MangaGenerateRequest{
		Prompt:  fullPrompt,
		Images:  request.Images,
		N:       generationCount,
		Size:    size,
		Quality: quality,
	})
	if err != nil {
		_, _ = s.store.AddCredits(c.Request().Context(), user.ID, creditsCost, "web_image_generation_refund", "web_image", "")
		return c.JSON(http.StatusBadGateway, models.ErrorResponse{Message: webImageErrorMessage(err)})
	}

	generated := make([]models.WebGeneratedImage, 0, len(images))
	for _, image := range images {
		if strings.TrimSpace(image) == "" {
			continue
		}
		item, err := s.persistGeneratedImage(c.Request().Context(), image)
		if err != nil {
			_, _ = s.store.AddCredits(c.Request().Context(), user.ID, creditsCost, "web_image_generation_refund", "web_image", "")
			return c.JSON(http.StatusBadGateway, models.ErrorResponse{Message: "保存生成图片失败"})
		}
		generated = append(generated, item)
	}
	if len(generated) == 0 {
		_, _ = s.store.AddCredits(c.Request().Context(), user.ID, creditsCost, "web_image_generation_refund", "web_image", "")
		return c.JSON(http.StatusBadGateway, models.ErrorResponse{Message: "生图服务没有返回图片"})
	}
	if _, err := s.store.CreateGalleryImages(c.Request().Context(), user.ID, prompt, strings.TrimSpace(request.Style), imageModel.ID, imageModel.Name, size, quality, imageModel.CreditCost, generated); err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "保存图库记录失败"})
	}
	user, _ = s.store.GetUser(c.Request().Context(), user.ID)
	responseImages, err := s.signGeneratedImages(c.Request().Context(), generated)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "生成图片访问链接失败"})
	}

	return c.JSON(http.StatusOK, models.WebImageGenerateResponse{
		Images:    responseImages,
		Prompt:    prompt,
		Style:     strings.TrimSpace(request.Style),
		ModelID:   imageModel.ID,
		ModelName: imageModel.Name,
		Size:      size,
		Quality:   quality,
		Credits:   creditsCost,
		User:      &user,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) createWebImageTask(c echo.Context) error {
	user, ok := s.requireUser(c)
	if !ok {
		return nil
	}
	var request models.WebImageGenerateRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "invalid image task payload"})
	}

	prompt := strings.TrimSpace(request.Prompt)
	if prompt == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "prompt is required"})
	}

	style := strings.TrimSpace(request.Style)
	size := normalizedImageSize(request.Size)
	quality := normalizedImageQuality(request.Quality)
	request.Size = size
	request.Quality = quality
	request.Style = style
	generationCount := normalizedGenerationCount(request.N)
	imageModel, err := s.resolveImageModel(c.Request().Context(), request.ModelID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "invalid image model"})
	}
	active, err := s.store.HasActiveWebImageTask(c.Request().Context(), user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "检查生成任务失败"})
	}
	if active {
		return c.JSON(http.StatusConflict, models.ErrorResponse{Message: "已有图片正在生成，请完成后再提交新的任务"})
	}
	creditsCost := generationCount * imageModel.CreditCost
	if _, err := s.store.ConsumeCredits(c.Request().Context(), user.ID, creditsCost, "web_image_task", "web_image_task", ""); errors.Is(err, store.ErrInsufficientCredits) {
		return c.JSON(http.StatusPaymentRequired, models.ErrorResponse{Message: "credits are not enough"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "consume credits failed"})
	}

	task, err := s.store.CreateWebImageTask(c.Request().Context(), user.ID, prompt, style, imageModel.ID, size, quality, generationCount, creditsCost)
	if errors.Is(err, store.ErrConflict) {
		_, _ = s.store.AddCredits(c.Request().Context(), user.ID, creditsCost, "web_image_task_refund", "web_image_task", "")
		return c.JSON(http.StatusConflict, models.ErrorResponse{Message: "已有图片正在生成，请完成后再提交新的任务"})
	}
	if err != nil {
		_, _ = s.store.AddCredits(c.Request().Context(), user.ID, creditsCost, "web_image_task_refund", "web_image_task", "")
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "create image task failed"})
	}

	imageRefs := append([]string(nil), request.Images...)
	go s.processWebImageTask(task, imageRefs, request)

	user, _ = s.store.GetUser(c.Request().Context(), user.ID)
	return c.JSON(http.StatusAccepted, models.WebImageTaskResponse{Task: task, User: &user})
}

func (s *Server) listWebImageTasks(c echo.Context) error {
	user, ok := s.requireUser(c)
	if !ok {
		return nil
	}
	limit := queryInt(c, "limit", 20)
	items, err := s.store.ListWebImageTasks(c.Request().Context(), user.ID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "load image tasks failed"})
	}
	for index := range items {
		items[index], _ = s.signWebImageTask(c.Request().Context(), items[index])
	}
	return c.JSON(http.StatusOK, models.WebImageTasksResponse{Items: items})
}

func (s *Server) getWebImageTask(c echo.Context) error {
	user, ok := s.requireUser(c)
	if !ok {
		return nil
	}
	task, err := s.store.GetWebImageTask(c.Request().Context(), user.ID, c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "image task not found"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "load image task failed"})
	}
	task, _ = s.signWebImageTask(c.Request().Context(), task)
	return c.JSON(http.StatusOK, models.WebImageTaskResponse{Task: task})
}

func (s *Server) setWebImageTaskPublic(c echo.Context) error {
	user, ok := s.requireUser(c)
	if !ok {
		return nil
	}
	var request models.WebImageTaskVisibilityRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "invalid task visibility payload"})
	}
	task, err := s.store.SetWebImageTaskPublic(c.Request().Context(), user.ID, c.Param("id"), request.IsPublic)
	if errors.Is(err, store.ErrNotFound) {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "image task not found"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "update image task visibility failed"})
	}
	task, _ = s.signWebImageTask(c.Request().Context(), task)
	return c.JSON(http.StatusOK, models.WebImageTaskResponse{Task: task})
}

func (s *Server) processWebImageTask(task models.WebImageTask, imageRefs []string, request models.WebImageGenerateRequest) {
	ctx := context.Background()
	if err := s.store.MarkWebImageTaskRunning(ctx, task.UserID, task.ID); err != nil {
		return
	}

	fullPrompt := buildWebImagePrompt(task.Prompt, task.Style, request)
	s.logWebImagePrompt("web image task prompt", task.ModelID, task.Style, task.Size, fullPrompt)
	images, err := s.callAinaibaImage(ctx, models.MangaGenerateRequest{
		Prompt:  fullPrompt,
		Images:  imageRefs,
		N:       task.N,
		Size:    task.Size,
		Quality: task.Quality,
	})
	if err != nil {
		_, _ = s.store.AddCredits(ctx, task.UserID, task.CreditsCost, "web_image_task_refund", "web_image_task", task.ID)
		_, _ = s.store.FailWebImageTask(ctx, task.UserID, task.ID, webImageErrorMessage(err))
		return
	}

	generated := make([]models.WebGeneratedImage, 0, len(images))
	for _, image := range images {
		if strings.TrimSpace(image) == "" {
			continue
		}
		item, err := s.persistGeneratedImage(ctx, image)
		if err != nil {
			_, _ = s.store.AddCredits(ctx, task.UserID, task.CreditsCost, "web_image_task_refund", "web_image_task", task.ID)
			_, _ = s.store.FailWebImageTask(ctx, task.UserID, task.ID, "save generated image failed")
			return
		}
		generated = append(generated, item)
	}
	if len(generated) == 0 {
		_, _ = s.store.AddCredits(ctx, task.UserID, task.CreditsCost, "web_image_task_refund", "web_image_task", task.ID)
		_, _ = s.store.FailWebImageTask(ctx, task.UserID, task.ID, "no image result returned")
		return
	}

	items, err := s.store.CreateGalleryImages(ctx, task.UserID, task.Prompt, task.Style, task.ModelID, task.ModelName, task.Size, task.Quality, task.CreditsCost/task.N, generated)
	if err != nil {
		_, _ = s.store.AddCredits(ctx, task.UserID, task.CreditsCost, "web_image_task_refund", "web_image_task", task.ID)
		_, _ = s.store.FailWebImageTask(ctx, task.UserID, task.ID, "save generated images failed")
		return
	}
	galleryID := ""
	if len(items) > 0 {
		galleryID = items[0].ID
	}
	_, _ = s.store.CompleteWebImageTask(ctx, task.UserID, task.ID, generated[0], galleryID)
}

func normalizedGenerationCount(n int) int {
	if n < 1 {
		return 1
	}
	if n > 4 {
		return 4
	}
	return n
}

func normalizedImageQuality(quality string) string {
	switch strings.ToLower(strings.TrimSpace(quality)) {
	case "high":
		return "high"
	case "low":
		return "low"
	case "standard", "标准", "medium":
		return "medium"
	default:
		return "medium"
	}
}

func normalizedImageSize(size string) string {
	value := strings.ToLower(strings.TrimSpace(size))
	switch value {
	case "", "auto", "自动":
		return "1024x1360"
	case "1:1":
		return "1024x1024"
	case "3:4":
		return "1024x1360"
	case "4:5":
		return "1024x1280"
	case "2:3":
		return "1024x1536"
	case "9:16":
		return "1024x1792"
	case "4:3":
		return "1360x1024"
	case "5:4":
		return "1280x1024"
	case "3:2":
		return "1536x1024"
	case "16:9":
		return "1792x1024"
	case "21:9":
		return "1792x768"
	}
	widthText, heightText, ok := strings.Cut(value, "x")
	if !ok {
		return "1024x1360"
	}
	width, widthErr := strconv.Atoi(strings.TrimSpace(widthText))
	height, heightErr := strconv.Atoi(strings.TrimSpace(heightText))
	if widthErr != nil || heightErr != nil {
		return "1024x1360"
	}
	width = normalizeImageDimension(width)
	height = normalizeImageDimension(height)
	return strconv.Itoa(width) + "x" + strconv.Itoa(height)
}

func normalizeImageDimension(value int) int {
	if value < 256 {
		return 256
	}
	if value > 2048 {
		value = 2048
	}
	rounded := (value / 16) * 16
	if rounded < 256 {
		return 256
	}
	return rounded
}

func (s *Server) listWebGallery(c echo.Context) error {
	limit := queryInt(c, "limit", 30)
	userID := ""
	isAuthed := false
	if user, err := s.currentUser(c); err == nil {
		userID = user.ID
		isAuthed = true
	}
	if !isAuthed && limit > 20 {
		limit = 20
	}
	var items []models.WebGalleryImage
	var err error
	if c.QueryParam("favorite") == "true" || c.QueryParam("favorites") == "true" {
		if !isAuthed {
			return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: "请先登录后查看收藏"})
		}
		items, err = s.store.ListFavoriteGalleryImages(c.Request().Context(), userID, limit, c.QueryParam("before"), c.QueryParam("q"))
	} else {
		items, err = s.store.ListGalleryImages(c.Request().Context(), userID, limit, c.QueryParam("before"), c.QueryParam("q"))
	}
	if err != nil {
		if s.logger != nil {
			s.logger.Error("load web gallery failed", "limit", limit, "before", c.QueryParam("before"), "error", err)
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "图库加载失败"})
	}
	if s.logger != nil {
		s.logger.Info("web gallery loaded", "limit", limit, "before", c.QueryParam("before"), "items", len(items))
	}
	items, err = s.signGalleryImages(c.Request().Context(), items)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "图片访问链接生成失败"})
	}
	return c.JSON(http.StatusOK, models.WebGalleryResponse{Items: items})
}

func (s *Server) listImageModels(c echo.Context) error {
	items, err := s.store.ListImageModels(c.Request().Context())
	if err != nil || len(items) == 0 {
		items = defaultImageModels()
	}
	return c.JSON(http.StatusOK, models.ImageModelsResponse{Items: items})
}

func (s *Server) likeWebGalleryImage(c echo.Context) error {
	user, ok := s.requireUser(c)
	if !ok {
		return nil
	}
	var request models.WebGalleryLikeRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "点赞请求格式不正确"})
	}
	item, err := s.store.SetGalleryImageLike(c.Request().Context(), user.ID, c.Param("id"), request.Liked)
	if errors.Is(err, store.ErrNotFound) {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "图片不存在"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "点赞更新失败"})
	}
	item, _ = s.signGalleryImage(c.Request().Context(), item)
	return c.JSON(http.StatusOK, models.WebGalleryLikeResponse{Item: item})
}

func (s *Server) favoriteWebGalleryImage(c echo.Context) error {
	user, ok := s.requireUser(c)
	if !ok {
		return nil
	}
	var request models.WebGalleryFavoriteRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "收藏请求格式不正确"})
	}
	item, err := s.store.SetGalleryImageFavorite(c.Request().Context(), user.ID, c.Param("id"), request.Favorited)
	if errors.Is(err, store.ErrNotFound) {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "图片不存在"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "收藏更新失败"})
	}
	item, _ = s.signGalleryImage(c.Request().Context(), item)
	return c.JSON(http.StatusOK, models.WebGalleryFavoriteResponse{Item: item})
}

func (s *Server) featureWebGalleryImage(c echo.Context) error {
	user, ok := s.requireUser(c)
	if !ok {
		return nil
	}
	var request models.WebGalleryFeaturedRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "精选请求格式不正确"})
	}
	item, err := s.store.SetGalleryImageFeatured(c.Request().Context(), user.ID, c.Param("id"), request.IsFeatured, request.IsPromptFeatured)
	if errors.Is(err, store.ErrNotFound) {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "图片不存在"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "精选更新失败"})
	}
	item, _ = s.signGalleryImage(c.Request().Context(), item)
	return c.JSON(http.StatusOK, models.WebGalleryFeaturedResponse{Item: item})
}

func (s *Server) resolveImageModel(ctx context.Context, modelID string) (models.ImageModel, error) {
	model, err := s.store.GetImageModel(ctx, modelID)
	if err == nil && model.CreditCost > 0 {
		return model, nil
	}
	if strings.TrimSpace(modelID) != "" {
		for _, candidate := range defaultImageModels() {
			if candidate.ID == strings.TrimSpace(modelID) {
				return candidate, nil
			}
		}
		return models.ImageModel{}, err
	}
	return defaultImageModels()[0], nil
}

func defaultImageModels() []models.ImageModel {
	return []models.ImageModel{
		{ID: "gpt-image", Name: "GPT Image", Provider: "OpenAI", Description: "通用商业海报与插画", CreditCost: webGenerationCreditCost, Enabled: true},
		{ID: "seedream", Name: "Seedream", Provider: "ByteDance", Description: "中文提示词友好", CreditCost: 6, Enabled: true},
		{ID: "qwen-image", Name: "Qwen Image", Provider: "Alibaba", Description: "产品图与中文排版", CreditCost: 5, Enabled: true},
		{ID: "gemini-image", Name: "Gemini Image", Provider: "Google", Description: "多模态参考图创作", CreditCost: 8, Enabled: true},
	}
}

func (s *Server) signGeneratedImages(ctx context.Context, images []models.WebGeneratedImage) ([]models.WebGeneratedImage, error) {
	signed := make([]models.WebGeneratedImage, 0, len(images))
	for _, image := range images {
		next := image
		if bucket, key, ok := parseOSSStorageURI(image.URL); ok {
			url, err := s.signedOSSObjectURL(ctx, bucket, key, "")
			if err != nil {
				return nil, err
			}
			thumbnailURL, err := s.signedOSSObjectURL(ctx, bucket, key, ossThumbnailProcess)
			if err != nil {
				return nil, err
			}
			next.URL = url
			next.ThumbnailURL = thumbnailURL
		}
		if bucket, key, ok := parseR2StorageURI(image.URL); ok {
			url, err := s.signedR2ObjectURL(ctx, bucket, key)
			if err != nil {
				return nil, err
			}
			next.URL = url
			next.ThumbnailURL = url
		}
		signed = append(signed, next)
	}
	return signed, nil
}

func (s *Server) signGalleryImages(ctx context.Context, items []models.WebGalleryImage) ([]models.WebGalleryImage, error) {
	signed := make([]models.WebGalleryImage, 0, len(items))
	for _, item := range items {
		next, err := s.signGalleryImage(ctx, item)
		if err != nil {
			return nil, err
		}
		signed = append(signed, next)
	}
	return signed, nil
}

func (s *Server) signGalleryImage(ctx context.Context, item models.WebGalleryImage) (models.WebGalleryImage, error) {
	if bucket, key, ok := parseOSSStorageURI(item.Image); ok {
		url, err := s.signedOSSObjectURL(ctx, bucket, key, "")
		if err != nil {
			return item, err
		}
		thumbnailURL, err := s.signedOSSObjectURL(ctx, bucket, key, ossThumbnailProcess)
		if err != nil {
			return item, err
		}
		item.Image = url
		item.ThumbnailURL = thumbnailURL
	}
	if bucket, key, ok := parseR2StorageURI(item.Image); ok {
		url, err := s.signedR2ObjectURL(ctx, bucket, key)
		if err != nil {
			return item, err
		}
		item.Image = url
		item.ThumbnailURL = url
	}
	if bucket, key, ok := parseOSSStorageURI(item.AuthorAvatarURL); ok {
		url, err := s.signedOSSObjectURL(ctx, bucket, key, "")
		if err != nil {
			return item, err
		}
		item.AuthorAvatarURL = url
	}
	if bucket, key, ok := parseR2StorageURI(item.AuthorAvatarURL); ok {
		url, err := s.signedR2ObjectURL(ctx, bucket, key)
		if err != nil {
			return item, err
		}
		item.AuthorAvatarURL = url
	}
	return item, nil
}

func (s *Server) signWebImageTask(ctx context.Context, task models.WebImageTask) (models.WebImageTask, error) {
	if bucket, key, ok := parseOSSStorageURI(task.ResultImage); ok {
		url, err := s.signedOSSObjectURL(ctx, bucket, key, "")
		if err != nil {
			return task, err
		}
		task.ResultImage = url
	}
	if bucket, key, ok := parseR2StorageURI(task.ResultImage); ok {
		url, err := s.signedR2ObjectURL(ctx, bucket, key)
		if err != nil {
			return task, err
		}
		task.ResultImage = url
	}
	return task, nil
}

func buildWebImagePrompt(prompt string, style string, request models.WebImageGenerateRequest) string {
	style = strings.TrimSpace(style)
	parts := []string{prompt}

	stylePrompts := map[string]string{
		"日漫风":  "Japanese anime and manga style, clean character design, expressive linework, cinematic composition.",
		"美漫风":  "American comic book style, bold inks, dynamic poses, graphic lighting, polished panel-ready composition.",
		"黑白漫画": "black-and-white manga style, crisp ink line art, screentone shading, strong contrast, no color except if essential.",
		"赛博朋克": "cyberpunk manga style, neon lighting, futuristic city atmosphere, high contrast red and cyan accents.",
		"国风漫画": "Chinese fantasy comic style, elegant ink-wash influence, flowing costume details, poetic atmosphere.",
		"奇幻冒险": "fantasy adventure manga style, epic sense of scale, magical lighting, detailed worldbuilding.",
	}
	if style != "" && style != "推荐" {
		if stylePrompt, ok := stylePrompts[style]; ok {
			parts = append(parts, "Style preset: "+stylePrompt)
		} else {
			parts = append(parts, "Style preset: "+style+". Keep the result polished, high quality, and suitable for a manga image studio gallery.")
		}
	}
	if size := strings.TrimSpace(request.Size); size != "" {
		parts = append(parts, "Canvas size: "+size+". Preserve the requested composition ratio.")
	}
	if strings.EqualFold(strings.TrimSpace(request.Resolution), "2k") {
		parts = append(parts, "Resolution target: crisp 2K-ready details with clean edges and readable fine texture.")
	}
	if request.LockedSeed {
		parts = append(parts, "Keep variations consistent with the current composition and character identity as much as possible.")
	}
	if strings.TrimSpace(request.NegativePrompt) != "" {
		parts = append(parts, "Avoid: "+strings.TrimSpace(request.NegativePrompt)+".")
	}
	if len(request.Images) > 0 {
		parts = append(parts, "Use the supplied reference image only for visual guidance. Keep the generated result original.")
	}
	return strings.Join(parts, "\n\n")
}

func (s *Server) logWebImagePrompt(message string, modelID string, style string, size string, prompt string) {
	if s.logger == nil {
		return
	}
	s.logger.Info(message, "modelID", modelID, "style", strings.TrimSpace(style), "size", size, "fullPrompt", prompt)
}

func webImageErrorMessage(err error) string {
	message := strings.TrimSpace(err.Error())
	lower := strings.ToLower(message)
	if strings.Contains(lower, "api key") || strings.Contains(lower, "api_key") || strings.Contains(lower, "authorization") || strings.Contains(lower, "bearer") {
		return "图像生成服务的 API Key 未配置或无效，请检查后端 XAI_API_KEY 配置"
	}
	if strings.Contains(lower, "no image result") {
		return "生图服务没有返回图片"
	}
	if strings.Contains(lower, "divisible by 16") || strings.Contains(lower, "invalid size") || (strings.Contains(lower, "invalid_value") && strings.Contains(lower, "size")) {
		return "图片尺寸不符合模型要求，系统已修正尺寸配置，请重新生成"
	}
	if message == "" {
		return "图像生成服务暂时不可用"
	}
	return message
}
