package httpapi

import (
	"strings"
	"testing"
)

func TestBuildEmailMessageIncludesBerserkVerificationCard(t *testing.T) {
	htmlBody := buildVerificationEmailHTML("123456", "注册", 90)
	message := buildEmailMessage(
		"noreply@example.com",
		"Berserk AI",
		"user@example.com",
		"Berserk AI 邮箱验证码",
		"你的 Berserk AI 注册验证码是：123456",
		htmlBody,
	)

	checks := []string{
		"Content-Type: multipart/alternative;",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Type: text/html; charset=UTF-8",
		"Berserk <span style=\"color:#8b35ff;\">AI</span>",
		"你的 Berserk AI 验证码",
		"123456",
		"验证码将在 <strong style=\"color:#17142a;\">90 秒</strong> 后失效",
	}
	for _, check := range checks {
		if !strings.Contains(message, check) {
			t.Fatalf("expected email message to contain %q", check)
		}
	}
}

func TestEmailCodeHashIgnoresAppIDForSingleService(t *testing.T) {
	left := emailCodeHash("mangaai.web", "USER@example.com", "register", "123456")
	right := emailCodeHash("berserk.web", "user@example.com", "register", "123456")
	if left != right {
		t.Fatalf("expected app id independent email code hash")
	}
	if left == legacyEmailCodeHash("mangaai.web", "USER@example.com", "register", "123456") {
		t.Fatalf("expected legacy hash to remain distinct")
	}
}

func TestSMTPPort465UsesImplicitTLS(t *testing.T) {
	server := &Server{
		smtpPort:    "465",
		smtpTLSMode: "starttls",
	}

	if got := server.normalizedSMTPTLSMode(); got != "tls" {
		t.Fatalf("expected port 465 to use implicit tls, got %q", got)
	}
}
