package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"pianke-ticket/backend/internal/models"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	osscredentials "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/aws/aws-sdk-go-v2/aws"
	awscredentials "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const maxGeneratedImageBytes = 16 << 20
const ossThumbnailProcess = "image/resize,w_640/quality,q_78/format,webp"
const defaultOSSSignedURLTTL = time.Hour

func (s *Server) persistGeneratedImage(ctx context.Context, image string) (models.WebGeneratedImage, error) {
	image = strings.TrimSpace(image)
	if image == "" {
		return models.WebGeneratedImage{}, errors.New("empty generated image")
	}

	var (
		data []byte
		mime = "image/png"
		err  error
	)
	if strings.HasPrefix(image, "http://") || strings.HasPrefix(image, "https://") {
		data, mime, err = downloadGeneratedImage(ctx, image)
	} else {
		data, mime, err = decodeGeneratedImage(image)
	}
	if err != nil {
		return models.WebGeneratedImage{}, err
	}
	if len(data) == 0 {
		return models.WebGeneratedImage{}, errors.New("empty generated image data")
	}
	if len(data) > maxGeneratedImageBytes {
		return models.WebGeneratedImage{}, errors.New("generated image is too large")
	}
	if !strings.HasPrefix(mime, "image/") {
		mime = http.DetectContentType(data)
	}
	if !strings.HasPrefix(mime, "image/") {
		return models.WebGeneratedImage{}, errors.New("generated payload is not an image")
	}

	filename, err := generatedImageFilename(mime)
	if err != nil {
		return models.WebGeneratedImage{}, err
	}

	if s.r2Configured() {
		objectKey := s.r2ObjectKey(filename)
		if err := s.uploadGeneratedImageToR2(ctx, objectKey, data, mime); err != nil {
			return models.WebGeneratedImage{}, err
		}
		return models.WebGeneratedImage{
			URL:      r2StorageURI(s.r2Bucket, objectKey),
			MimeType: mime,
		}, nil
	}

	if s.ossConfigured() {
		objectKey := s.ossObjectKey(filename)
		if err := s.uploadGeneratedImageToOSS(ctx, objectKey, data, mime); err != nil {
			return models.WebGeneratedImage{}, err
		}
		return models.WebGeneratedImage{
			URL:      ossStorageURI(s.ossBucket, objectKey),
			MimeType: mime,
		}, nil
	}

	if err := os.MkdirAll(s.generatedImageDir, 0o755); err != nil {
		return models.WebGeneratedImage{}, err
	}
	path := filepath.Join(s.generatedImageDir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return models.WebGeneratedImage{}, err
	}
	return models.WebGeneratedImage{
		URL:          s.generatedImageURL(filename),
		ThumbnailURL: "",
		MimeType:     mime,
	}, nil
}

func decodeGeneratedImage(image string) ([]byte, string, error) {
	mime := "image/png"
	payload := image
	if strings.HasPrefix(image, "data:") {
		header, body, ok := strings.Cut(image, ",")
		if !ok {
			return nil, "", errors.New("invalid data URL image")
		}
		payload = body
		meta := strings.TrimPrefix(header, "data:")
		if detected, _, ok := strings.Cut(meta, ";"); ok && strings.HasPrefix(detected, "image/") {
			mime = detected
		}
	}
	payload = strings.NewReplacer("\n", "", "\r", "", "\t", "", " ", "").Replace(payload)
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(payload)
	}
	if err != nil {
		data, err = base64.URLEncoding.DecodeString(payload)
	}
	if err != nil {
		return nil, "", err
	}
	if detected := http.DetectContentType(data); strings.HasPrefix(detected, "image/") {
		mime = detected
	}
	return data, mime, nil
}

func downloadGeneratedImage(ctx context.Context, url string) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", errors.New("download generated image failed")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxGeneratedImageBytes+1))
	if err != nil {
		return nil, "", err
	}
	mime := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	if !strings.HasPrefix(mime, "image/") {
		mime = http.DetectContentType(data)
	}
	return data, mime, nil
}

func generatedImageFilename(mime string) (string, error) {
	var seed [16]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return "", err
	}
	ext := ".png"
	switch strings.ToLower(mime) {
	case "image/jpeg", "image/jpg":
		ext = ".jpg"
	case "image/webp":
		ext = ".webp"
	case "image/gif":
		ext = ".gif"
	}
	return time.Now().UTC().Format("20060102-150405-") + hex.EncodeToString(seed[:]) + ext, nil
}

func (s *Server) generatedImageURL(filename string) string {
	base := strings.TrimRight(strings.TrimSpace(s.publicBaseURL), "/")
	if base == "" {
		return "/berserk/generated/" + filename
	}
	return base + "/generated/" + filename
}

func (s *Server) r2Configured() bool {
	return strings.TrimSpace(s.r2Bucket) != "" &&
		strings.TrimSpace(s.r2Endpoint) != "" &&
		strings.TrimSpace(s.r2AccessKeyID) != "" &&
		strings.TrimSpace(s.r2AccessKeySecret) != ""
}

func (s *Server) uploadGeneratedImageToR2(ctx context.Context, objectKey string, data []byte, mime string) error {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	contentLength := int64(len(data))
	_, err := s.r2Client().PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.r2Bucket),
		Key:           aws.String(objectKey),
		Body:          bytes.NewReader(data),
		ContentLength: &contentLength,
		ContentType:   aws.String(mime),
		CacheControl:  aws.String("private, max-age=3600"),
	})
	return err
}

func (s *Server) r2ObjectKey(filename string) string {
	prefix := strings.Trim(strings.TrimSpace(s.r2ObjectPrefix), "/")
	if prefix == "" {
		return filename
	}
	return prefix + "/" + filename
}

func (s *Server) r2Client() *s3.Client {
	return s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(s.r2Endpoint),
		Credentials: aws.NewCredentialsCache(
			awscredentials.NewStaticCredentialsProvider(s.r2AccessKeyID, s.r2AccessKeySecret, ""),
		),
	})
}

func (s *Server) signedR2ObjectURL(ctx context.Context, bucket string, objectKey string) (string, error) {
	if !s.r2Configured() {
		return "", errors.New("r2 is not configured")
	}
	bucket = firstNonEmpty(strings.TrimSpace(bucket), s.r2Bucket)
	objectKey = strings.TrimLeft(strings.TrimSpace(objectKey), "/")
	if bucket == "" || objectKey == "" {
		return "", errors.New("invalid r2 object")
	}
	presigner := s3.NewPresignClient(s.r2Client())
	result, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	}, s3.WithPresignExpires(s.r2SignedURLTTL))
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

func (s *Server) ossConfigured() bool {
	return strings.TrimSpace(s.ossBucket) != "" &&
		strings.TrimSpace(s.ossRegion) != "" &&
		strings.TrimSpace(s.ossEndpoint) != "" &&
		strings.TrimSpace(s.ossAccessKeyID) != "" &&
		strings.TrimSpace(s.ossAccessKeySecret) != ""
}

func (s *Server) uploadGeneratedImageToOSS(ctx context.Context, objectKey string, data []byte, mime string) error {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	client := s.ossClient()
	contentLength := int64(len(data))
	_, err := client.PutObject(ctx, &oss.PutObjectRequest{
		Bucket:          oss.Ptr(s.ossBucket),
		Key:             oss.Ptr(objectKey),
		Body:            bytes.NewReader(data),
		ContentLength:   oss.Ptr(contentLength),
		ContentType:     oss.Ptr(mime),
		CacheControl:    oss.Ptr("public, max-age=31536000, immutable"),
		ForbidOverwrite: oss.Ptr("true"),
	})
	return err
}

func (s *Server) ossObjectKey(filename string) string {
	prefix := strings.Trim(strings.TrimSpace(s.ossObjectPrefix), "/")
	if prefix == "" {
		return filename
	}
	return prefix + "/" + filename
}

func (s *Server) ossClient() *oss.Client {
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(osscredentials.NewStaticCredentialsProvider(s.ossAccessKeyID, s.ossAccessKeySecret, s.ossSecurityToken)).
		WithRegion(normalizeOSSRegion(s.ossRegion)).
		WithEndpoint(s.ossEndpoint)
	return oss.NewClient(cfg)
}

func (s *Server) signedOSSObjectURL(ctx context.Context, bucket string, objectKey string, process string) (string, error) {
	if !s.ossConfigured() {
		return "", errors.New("oss is not configured")
	}
	bucket = firstNonEmpty(strings.TrimSpace(bucket), s.ossBucket)
	objectKey = strings.TrimLeft(strings.TrimSpace(objectKey), "/")
	if bucket == "" || objectKey == "" {
		return "", errors.New("invalid oss object")
	}
	request := &oss.GetObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(objectKey),
	}
	if strings.TrimSpace(process) != "" {
		request.Process = oss.Ptr(process)
	}
	result, err := s.ossClient().Presign(ctx, request, oss.PresignExpires(s.ossSignedURLTTL))
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

func signedURLTTL(value string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds <= 0 {
		return defaultOSSSignedURLTTL
	}
	return time.Duration(seconds) * time.Second
}

func normalizeOSSRegion(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "oss-") {
		return strings.TrimPrefix(value, "oss-")
	}
	return value
}

func normalizeR2Endpoint(value string) string {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return "https://" + value
}

func ossStorageURI(bucket string, objectKey string) string {
	return (&url.URL{
		Scheme: "oss",
		Host:   strings.TrimSpace(bucket),
		Path:   "/" + strings.TrimLeft(strings.TrimSpace(objectKey), "/"),
	}).String()
}

func parseOSSStorageURI(value string) (string, string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "oss" || parsed.Host == "" {
		return "", "", false
	}
	key := strings.TrimLeft(parsed.Path, "/")
	if key == "" {
		return "", "", false
	}
	return parsed.Host, key, true
}

func r2StorageURI(bucket string, objectKey string) string {
	return (&url.URL{
		Scheme: "r2",
		Host:   strings.TrimSpace(bucket),
		Path:   "/" + strings.TrimLeft(strings.TrimSpace(objectKey), "/"),
	}).String()
}

func parseR2StorageURI(value string) (string, string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "r2" || parsed.Host == "" {
		return "", "", false
	}
	key := strings.TrimLeft(parsed.Path, "/")
	if key == "" {
		return "", "", false
	}
	return parsed.Host, key, true
}
