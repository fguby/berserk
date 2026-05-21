package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	AppEnv                 string
	HTTPAddr               string
	DatabaseURL            string
	PublicBaseURL          string
	WebAppID               string
	EmailCodeTTLSeconds    string
	SMTPHost               string
	SMTPPort               string
	SMTPUsername           string
	SMTPPassword           string
	SMTPFromEmail          string
	SMTPFromName           string
	SMTPTLSMode            string
	XAIAPIKey              string
	XAIBaseURL             string
	XAIResponsesPath       string
	XAIMainModel           string
	XAIImageModel          string
	GeneratedImageDir      string
	OSSBucket              string
	OSSRegion              string
	OSSEndpoint            string
	OSSAccessKeyID         string
	OSSAccessKeySecret     string
	OSSSecurityToken       string
	OSSObjectPrefix        string
	OSSSignedURLTTLSeconds string
	R2Bucket               string
	R2Endpoint             string
	R2AccessKeyID          string
	R2AccessKeySecret      string
	R2ObjectPrefix         string
	R2SignedURLTTLSeconds  string
}

func Load() Config {
	fileConfig := loadFileConfig(getEnv("CONFIG_PATH", "config.yaml"))
	return Config{
		AppEnv:                 getEnv("APP_ENV", fileConfig.value("app_env", "development")),
		HTTPAddr:               getEnv("HTTP_ADDR", fileConfig.value("http_addr", ":8080")),
		DatabaseURL:            getEnv("DATABASE_URL", fileConfig.value("database_url", "postgres://berserk:berserk@localhost:5432/berserk?sslmode=disable")),
		PublicBaseURL:          getEnv("PUBLIC_BASE_URL", fileConfig.value("public_base_url", "https://www.eatfit.fun")),
		WebAppID:               getEnv("WEB_APP_ID", fileConfig.value("web_app_id", "berserk.web")),
		EmailCodeTTLSeconds:    getEnv("EMAIL_CODE_TTL_SECONDS", fileConfig.value("email_code_ttl_seconds", "90")),
		SMTPHost:               getEnv("SMTP_HOST", fileConfig.value("smtp_host", "")),
		SMTPPort:               getEnv("SMTP_PORT", fileConfig.value("smtp_port", "587")),
		SMTPUsername:           getEnv("SMTP_USERNAME", fileConfig.value("smtp_username", "")),
		SMTPPassword:           getEnv("SMTP_PASSWORD", fileConfig.value("smtp_password", "")),
		SMTPFromEmail:          getEnv("SMTP_FROM_EMAIL", fileConfig.value("smtp_from_email", "")),
		SMTPFromName:           getEnv("SMTP_FROM_NAME", fileConfig.value("smtp_from_name", "Berserk AI")),
		SMTPTLSMode:            getEnv("SMTP_TLS_MODE", fileConfig.value("smtp_tls_mode", "starttls")),
		XAIAPIKey:              getEnv("XAI_API_KEY", fileConfig.value("xai_api_key", "")),
		XAIBaseURL:             strings.TrimRight(getEnv("XAI_BASE_URL", fileConfig.value("xai_base_url", "https://api-xai.ainaibahub.com/v1")), "/"),
		XAIResponsesPath:       getEnv("XAI_RESPONSES_PATH", fileConfig.value("xai_responses_path", "/responses")),
		XAIMainModel:           getEnv("XAI_MAIN_MODEL", fileConfig.value("xai_main_model", "gpt-5.5")),
		XAIImageModel:          getEnv("XAI_IMAGE_MODEL", fileConfig.value("xai_image_model", "gpt-image-2")),
		GeneratedImageDir:      getEnv("GENERATED_IMAGE_DIR", fileConfig.value("generated_image_dir", "generated-images")),
		OSSBucket:              getEnv("OSS_BUCKET", fileConfig.value("oss_bucket", "")),
		OSSRegion:              getEnv("OSS_REGION", fileConfig.value("oss_region", "")),
		OSSEndpoint:            getEnv("OSS_ENDPOINT", fileConfig.value("oss_endpoint", "")),
		OSSAccessKeyID:         getEnv("OSS_ACCESS_KEY_ID", fileConfig.value("oss_access_key_id", "")),
		OSSAccessKeySecret:     getEnv("OSS_ACCESS_KEY_SECRET", fileConfig.value("oss_access_key_secret", "")),
		OSSSecurityToken:       getEnv("OSS_SECURITY_TOKEN", fileConfig.value("oss_security_token", "")),
		OSSObjectPrefix:        getEnv("OSS_OBJECT_PREFIX", fileConfig.value("oss_object_prefix", "berserk/generated")),
		OSSSignedURLTTLSeconds: getEnv("OSS_SIGNED_URL_TTL_SECONDS", fileConfig.value("oss_signed_url_ttl_seconds", "3600")),
		R2Bucket:               getEnv("R2_BUCKET", fileConfig.value("r2_bucket", "")),
		R2Endpoint:             getEnv("R2_ENDPOINT", fileConfig.value("r2_endpoint", "")),
		R2AccessKeyID:          getEnv("R2_ACCESS_KEY_ID", fileConfig.value("r2_access_key_id", "")),
		R2AccessKeySecret:      getEnv("R2_ACCESS_KEY_SECRET", fileConfig.value("r2_access_key_secret", "")),
		R2ObjectPrefix:         getEnv("R2_OBJECT_PREFIX", fileConfig.value("r2_object_prefix", "berserk/generated")),
		R2SignedURLTTLSeconds:  getEnv("R2_SIGNED_URL_TTL_SECONDS", fileConfig.value("r2_signed_url_ttl_seconds", "3600")),
	}
}

type fileConfig map[string]string

func loadFileConfig(path string) fileConfig {
	file, err := os.Open(path)
	if err != nil {
		return fileConfig{}
	}
	defer file.Close()

	values := fileConfig{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(strings.SplitN(value, "#", 2)[0])
		value = strings.Trim(value, `"'`)
		if key != "" {
			values[key] = value
		}
	}
	return values
}

func (cfg fileConfig) value(key string, fallback string) string {
	if value := cfg[key]; value != "" {
		return value
	}
	return fallback
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
