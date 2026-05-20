package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"math/big"
	"mime"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"pianke-ticket/backend/internal/models"
	"pianke-ticket/backend/internal/store"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultEmailCodeTTL       = 90 * time.Second
	defaultEmailSetupTokenTTL = 10 * time.Minute
)

func (s *Server) sendEmailCode(c echo.Context) error {
	var request models.EmailCodeRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "验证码请求格式不正确"})
	}
	email, ok := normalizeEmailAddress(request.Email)
	if !ok {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "邮箱地址不正确"})
	}
	mode := emailAuthMode(request.Mode)
	appID := s.emailAuthAppID(request.AppID)

	switch mode {
	case "register":
		if _, err := s.store.GetEmailUser(c.Request().Context(), appID, email); err == nil {
			return c.JSON(http.StatusConflict, models.ErrorResponse{Message: "该邮箱已经注册"})
		} else if !errors.Is(err, store.ErrNotFound) {
			return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "邮箱状态检查失败"})
		}
	case "login", "reset":
		if _, err := s.store.GetEmailUser(c.Request().Context(), appID, email); errors.Is(err, store.ErrNotFound) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "该邮箱尚未注册"})
		} else if err != nil {
			return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "邮箱状态检查失败"})
		}
	default:
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "邮箱验证类型不正确"})
	}

	code, err := randomEmailCode()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "验证码生成失败"})
	}
	expiresAt := time.Now().Add(s.emailCodeTTL)
	codeHash := emailCodeHash(appID, email, mode, code)
	if err := s.store.SaveEmailCode(c.Request().Context(), appID, email, mode, codeHash, expiresAt); err != nil {
		if s.logger != nil {
			s.logger.Error("save email code failed", "email", email, "mode", mode, "error", err)
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "验证码保存失败"})
	}
	if err := s.sendVerificationEmail(email, code, mode); err != nil {
		if s.logger != nil {
			s.logger.Error("send verification email failed", "email", email, "mode", mode, "error", err)
		}
		return c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{Message: "验证码发送失败，请检查邮箱服务配置后重试"})
	}
	return c.JSON(http.StatusOK, models.EmailCodeResponse{
		ExpiresIn: int(s.emailCodeTTL.Seconds()),
		ExpiresAt: expiresAt.UTC().Format(time.RFC3339),
	})
}

func (s *Server) authEmailRegister(c echo.Context) error {
	var request models.EmailSetupPasswordRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "注册请求格式不正确"})
	}
	email, ok := normalizeEmailAddress(request.Email)
	if !ok {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "邮箱地址不正确"})
	}
	passwordHash, ok := passwordHashForRequest(request.Password)
	if !ok {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "密码至少需要 8 位"})
	}
	appID := s.emailAuthAppID(request.AppID)
	setupToken := strings.TrimSpace(request.SetupToken)
	code := strings.TrimSpace(request.Code)
	var verifyErr error
	if code != "" {
		verifyErr = s.consumeEmailCodeWithFallback(c.Request().Context(), appID, email, "register", code)
	} else if setupToken != "" {
		verifyErr = s.consumeVerifiedEmailCodeWithFallback(c.Request().Context(), appID, email, "register", setupToken)
	} else {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "请填写邮箱验证码"})
	}
	if verifyErr != nil {
		if s.logger != nil {
			s.logger.Warn("register email code rejected", "email", email, "hasSetupToken", setupToken != "", "error", verifyErr)
		}
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: "验证码无效或已过期，请重新获取"})
	}
	user, err := s.store.CreateEmailUser(c.Request().Context(), appID, email, passwordHash)
	if errors.Is(err, store.ErrConflict) {
		return c.JSON(http.StatusConflict, models.ErrorResponse{Message: "该邮箱已经注册"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "注册失败，请稍后重试"})
	}
	token, err := s.store.CreateSession(c.Request().Context(), user.ID, time.Now().Add(90*24*time.Hour))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "登录状态创建失败"})
	}
	return c.JSON(http.StatusOK, models.AuthResponse{Token: token, User: user})
}

func (s *Server) verifyEmailCode(c echo.Context) error {
	var request models.EmailCodeVerifyRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "验证码请求格式不正确"})
	}
	email, ok := normalizeEmailAddress(request.Email)
	if !ok {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "邮箱地址不正确"})
	}
	code := strings.TrimSpace(request.Code)
	if code == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "请填写邮箱验证码"})
	}
	mode := emailAuthMode(request.Mode)
	if mode != "register" && mode != "reset" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "邮箱验证类型不正确"})
	}
	appID := s.emailAuthAppID(request.AppID)
	setupToken, err := randomHexToken(32)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "验证令牌生成失败"})
	}
	expiresAt := time.Now().Add(defaultEmailSetupTokenTTL)
	if err := s.verifyEmailCodeWithFallback(c.Request().Context(), appID, email, mode, code, setupToken, expiresAt); err != nil {
		if s.logger != nil {
			s.logger.Warn("email code verify rejected", "email", email, "mode", mode, "error", err)
		}
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: "验证码无效或已过期，请重新获取"})
	}
	return c.JSON(http.StatusOK, models.EmailCodeVerifyResponse{
		SetupToken: setupToken,
		ExpiresIn:  int(defaultEmailSetupTokenTTL.Seconds()),
		ExpiresAt:  expiresAt.UTC().Format(time.RFC3339),
	})
}

func (s *Server) authEmailLogin(c echo.Context) error {
	var request models.EmailPasswordLoginRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "登录请求格式不正确"})
	}
	email, ok := normalizeEmailAddress(request.Email)
	if !ok {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "邮箱地址不正确"})
	}
	password := strings.TrimSpace(request.Password)
	appID := s.emailAuthAppID(request.AppID)
	code := strings.TrimSpace(request.Code)
	if code != "" {
		user, err := s.store.GetEmailUser(c.Request().Context(), appID, email)
		if errors.Is(err, store.ErrNotFound) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "该邮箱尚未注册"})
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "登录失败，请稍后重试"})
		}
		if err := s.consumeEmailCodeWithFallback(c.Request().Context(), appID, email, "login", code); err != nil {
			return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: "验证码无效或已过期，请重新获取"})
		}
		token, err := s.store.CreateSession(c.Request().Context(), user.ID, time.Now().Add(90*24*time.Hour))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "登录状态创建失败"})
		}
		return c.JSON(http.StatusOK, models.AuthResponse{Token: token, User: user})
	}
	if password == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "请输入密码或邮箱验证码"})
	}
	passwordHash, err := s.store.GetEmailPasswordHash(c.Request().Context(), appID, email)
	if errors.Is(err, store.ErrNotFound) {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "该邮箱尚未注册"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "登录失败，请稍后重试"})
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: "邮箱或密码不正确"})
	}
	user, err := s.store.GetEmailUser(c.Request().Context(), appID, email)
	if errors.Is(err, store.ErrNotFound) {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "该邮箱尚未注册"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "登录失败，请稍后重试"})
	}
	token, err := s.store.CreateSession(c.Request().Context(), user.ID, time.Now().Add(90*24*time.Hour))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "登录状态创建失败"})
	}
	return c.JSON(http.StatusOK, models.AuthResponse{Token: token, User: user})
}

func (s *Server) resetEmailPassword(c echo.Context) error {
	var request models.EmailSetupPasswordRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "重置密码请求格式不正确"})
	}
	email, ok := normalizeEmailAddress(request.Email)
	if !ok {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "邮箱地址不正确"})
	}
	setupToken := strings.TrimSpace(request.SetupToken)
	if setupToken == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "请先完成邮箱验证"})
	}
	passwordHash, ok := passwordHashForRequest(request.Password)
	if !ok {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "密码至少需要 8 位"})
	}
	appID := s.emailAuthAppID(request.AppID)
	if err := s.consumeVerifiedEmailCodeWithFallback(c.Request().Context(), appID, email, "reset", setupToken); err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: "邮箱验证已过期，请重新获取验证码"})
	}
	if err := s.store.SetEmailUserPassword(c.Request().Context(), appID, email, passwordHash); errors.Is(err, store.ErrNotFound) {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "该邮箱尚未注册"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "重置密码失败，请稍后重试"})
	}
	user, err := s.store.GetEmailUser(c.Request().Context(), appID, email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "用户信息加载失败"})
	}
	token, err := s.store.CreateSession(c.Request().Context(), user.ID, time.Now().Add(90*24*time.Hour))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "登录状态创建失败"})
	}
	return c.JSON(http.StatusOK, models.AuthResponse{Token: token, User: user})
}

func (s *Server) emailAuthAppID(requestAppID string) string {
	return firstNonEmpty(s.webAppID, "berserk.web")
}

func emailAuthMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		return "register"
	}
	return mode
}

func passwordHashForRequest(password string) (string, bool) {
	password = strings.TrimSpace(password)
	if len(password) < 8 {
		return "", false
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", false
	}
	return string(hash), true
}

func emailCodeTTL(value string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds <= 0 {
		return defaultEmailCodeTTL
	}
	return time.Duration(seconds) * time.Second
}

func normalizeEmailAddress(value string) (string, bool) {
	address, err := mail.ParseAddress(strings.TrimSpace(value))
	if err != nil {
		return "", false
	}
	email := strings.ToLower(strings.TrimSpace(address.Address))
	return email, strings.Contains(email, "@")
}

func randomEmailCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func randomHexToken(size int) (string, error) {
	token := make([]byte, size)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return hex.EncodeToString(token), nil
}

func emailCodeHash(appID string, email string, purpose string, code string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.ToLower(strings.TrimSpace(email)),
		strings.ToLower(strings.TrimSpace(purpose)),
		strings.TrimSpace(code),
	}, "|")))
	return hex.EncodeToString(sum[:])
}

func emailSetupTokenHash(appID string, email string, purpose string, token string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.ToLower(strings.TrimSpace(email)),
		strings.ToLower(strings.TrimSpace(purpose)),
		strings.TrimSpace(token),
	}, "|")))
	return hex.EncodeToString(sum[:])
}

func legacyEmailCodeHash(appID string, email string, purpose string, code string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(appID),
		strings.ToLower(strings.TrimSpace(email)),
		strings.ToLower(strings.TrimSpace(purpose)),
		strings.TrimSpace(code),
	}, "|")))
	return hex.EncodeToString(sum[:])
}

func legacyEmailSetupTokenHash(appID string, email string, purpose string, token string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(appID),
		strings.ToLower(strings.TrimSpace(email)),
		strings.ToLower(strings.TrimSpace(purpose)),
		strings.TrimSpace(token),
	}, "|")))
	return hex.EncodeToString(sum[:])
}

func (s *Server) legacyEmailAppIDs(appID string) []string {
	candidates := []string{appID, s.webAppID, "berserk.web", "mangaai.web", "managai.web"}
	seen := map[string]bool{}
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		clean := strings.TrimSpace(candidate)
		if clean == "" || seen[clean] {
			continue
		}
		seen[clean] = true
		ids = append(ids, clean)
	}
	return ids
}

func (s *Server) verifyEmailCodeWithFallback(ctx context.Context, appID string, email string, purpose string, code string, setupToken string, expiresAt time.Time) error {
	setupHash := emailSetupTokenHash(appID, email, purpose, setupToken)
	if err := s.store.VerifyEmailCode(ctx, appID, email, purpose, emailCodeHash(appID, email, purpose, code), setupHash, expiresAt); err == nil {
		return nil
	}
	var lastErr error
	for _, legacyAppID := range s.legacyEmailAppIDs(appID) {
		if err := s.store.VerifyEmailCode(ctx, appID, email, purpose, legacyEmailCodeHash(legacyAppID, email, purpose, code), setupHash, expiresAt); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return lastErr
}

func (s *Server) consumeEmailCodeWithFallback(ctx context.Context, appID string, email string, purpose string, code string) error {
	if err := s.store.ConsumeEmailCode(ctx, appID, email, purpose, emailCodeHash(appID, email, purpose, code)); err == nil {
		return nil
	}
	var lastErr error
	for _, legacyAppID := range s.legacyEmailAppIDs(appID) {
		if err := s.store.ConsumeEmailCode(ctx, appID, email, purpose, legacyEmailCodeHash(legacyAppID, email, purpose, code)); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return lastErr
}

func (s *Server) consumeVerifiedEmailCodeWithFallback(ctx context.Context, appID string, email string, purpose string, setupToken string) error {
	if err := s.store.ConsumeVerifiedEmailCode(ctx, appID, email, purpose, emailSetupTokenHash(appID, email, purpose, setupToken)); err == nil {
		return nil
	}
	var lastErr error
	for _, legacyAppID := range s.legacyEmailAppIDs(appID) {
		if err := s.store.ConsumeVerifiedEmailCode(ctx, appID, email, purpose, legacyEmailSetupTokenHash(legacyAppID, email, purpose, setupToken)); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return lastErr
}

func (s *Server) sendVerificationEmail(email string, code string, mode string) error {
	if strings.TrimSpace(s.smtpHost) == "" || strings.TrimSpace(s.smtpFromEmail) == "" {
		return errors.New("smtp is not configured")
	}

	subject := "Berserk AI 邮箱验证码"
	action := "登录"
	if mode == "register" {
		action = "注册"
	} else if mode == "reset" {
		action = "重置密码"
	}
	body := fmt.Sprintf("你的 Berserk AI %s验证码是：%s\n\n验证码 %d 秒内有效，请勿转发给他人。", action, code, int(s.emailCodeTTL.Seconds()))
	htmlBody := buildVerificationEmailHTML(code, action, int(s.emailCodeTTL.Seconds()))
	message := buildEmailMessage(s.smtpFromEmail, "Berserk AI", email, subject, body, htmlBody)
	return s.sendSMTP(email, []byte(message))
}

func buildEmailMessage(fromEmail string, fromName string, to string, subject string, textBody string, htmlBody string) string {
	from := mail.Address{Name: fromName, Address: fromEmail}
	boundary := "berserk-email-boundary"
	if strings.TrimSpace(htmlBody) == "" {
		return strings.Join([]string{
			"From: " + from.String(),
			"To: " + to,
			"Subject: " + mime.QEncoding.Encode("UTF-8", subject),
			"MIME-Version: 1.0",
			"Content-Type: text/plain; charset=UTF-8",
			"Content-Transfer-Encoding: 8bit",
			"",
			textBody,
		}, "\r\n")
	}

	return strings.Join([]string{
		"From: " + from.String(),
		"To: " + to,
		"Subject: " + mime.QEncoding.Encode("UTF-8", subject),
		"MIME-Version: 1.0",
		`Content-Type: multipart/alternative; boundary="` + boundary + `"`,
		"",
		"--" + boundary,
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		textBody,
		"--" + boundary,
		"Content-Type: text/html; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		htmlBody,
		"--" + boundary + "--",
	}, "\r\n")
}

func buildVerificationEmailHTML(code string, action string, ttlSeconds int) string {
	code = html.EscapeString(code)
	action = html.EscapeString(action)
	return fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Berserk AI 邮箱验证码</title>
</head>
<body style="margin:0;padding:0;background:#f4f1ff;color:#17142a;font-family:-apple-system,BlinkMacSystemFont,'PingFang SC','Microsoft YaHei',Arial,sans-serif;">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:linear-gradient(180deg,#f4f1ff,#fbfaff);padding:34px 14px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="max-width:520px;border-collapse:separate;border-spacing:0;">
          <tr>
            <td style="padding:0 12px 10px;">
              <div style="font-size:24px;font-weight:950;letter-spacing:-.02em;color:#17142a;">Berserk <span style="color:#8b35ff;">AI</span></div>
            </td>
          </tr>
          <tr>
            <td style="position:relative;border:1px solid #ded3ff;border-radius:22px;background:#ffffff;box-shadow:0 22px 60px rgba(92,53,180,.16);overflow:hidden;">
              <div style="height:10px;background:linear-gradient(90deg,#5f00f5,#b800ff,#6c7bff);"></div>
              <div style="padding:32px 30px 28px;background-image:radial-gradient(circle at 92%% 8%%,rgba(184,0,255,.14),transparent 30%%),radial-gradient(circle at 8%% 92%%,rgba(95,0,245,.12),transparent 28%%);">
                <div style="display:inline-block;padding:8px 13px;border-radius:999px;background:#f1e9ff;color:#7c35ff;font-size:12px;font-weight:900;">%s验证码</div>
                <h1 style="margin:20px 0 8px;font-size:29px;line-height:1.15;font-weight:950;letter-spacing:0;color:#17142a;">你的 Berserk AI 验证码</h1>
                <p style="margin:0 0 22px;color:#635b75;font-size:14px;line-height:1.7;">把下面 6 位数字填回页面，继续完成%s。请勿转发给他人。</p>
                <div style="border-radius:18px;background:linear-gradient(135deg,#5f00f5,#b800ff);padding:2px;box-shadow:0 16px 34px rgba(124,53,255,.22);">
                  <div style="border-radius:16px;background:#fbfaff;padding:24px 18px;text-align:center;">
                    <div style="font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;font-size:44px;line-height:1;font-weight:950;letter-spacing:.2em;color:#4b18c8;text-indent:.2em;">%s</div>
                  </div>
                </div>
                <div style="margin-top:22px;padding:14px 16px;border-left:5px solid #8b35ff;border-radius:12px;background:#f7f2ff;color:#635b75;font-size:13px;line-height:1.65;">
                  验证码将在 <strong style="color:#17142a;">%d 秒</strong> 后失效。如果不是你本人操作，可以忽略这封邮件。
                </div>
              </div>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, action, action, code, ttlSeconds)
}

func (s *Server) sendSMTP(to string, message []byte) error {
	host := strings.TrimSpace(s.smtpHost)
	addr := net.JoinHostPort(host, firstNonEmpty(s.smtpPort, "587"))
	var auth smtp.Auth
	if strings.TrimSpace(s.smtpUsername) != "" {
		auth = smtp.PlainAuth("", s.smtpUsername, s.smtpPassword, host)
	}

	switch s.normalizedSMTPTLSMode() {
	case "tls":
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
		if err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			_ = conn.Close()
			return err
		}
		return sendSMTPWithClient(client, auth, s.smtpFromEmail, to, message)
	case "none":
		client, err := smtp.Dial(addr)
		if err != nil {
			return err
		}
		return sendSMTPWithClient(client, auth, s.smtpFromEmail, to, message)
	default:
		client, err := smtp.Dial(addr)
		if err != nil {
			return err
		}
		if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
			_ = client.Close()
			return err
		}
		return sendSMTPWithClient(client, auth, s.smtpFromEmail, to, message)
	}
}

func (s *Server) normalizedSMTPTLSMode() string {
	tlsMode := strings.ToLower(strings.TrimSpace(s.smtpTLSMode))
	if strings.TrimSpace(s.smtpPort) == "465" && tlsMode != "none" {
		return "tls"
	}
	return tlsMode
}

func sendSMTPWithClient(client *smtp.Client, auth smtp.Auth, from string, to string, message []byte) error {
	defer func() {
		_ = client.Close()
	}()
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
