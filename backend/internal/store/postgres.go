package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"pianke-ticket/backend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")
var ErrInsufficientCredits = errors.New("insufficient credits")

type BerserkStore interface {
	CreateEmailUser(ctx context.Context, appID string, email string, passwordHash string) (models.User, error)
	GetEmailUser(ctx context.Context, appID string, email string) (models.User, error)
	GetEmailPasswordHash(ctx context.Context, appID string, email string) (string, error)
	SetEmailUserPassword(ctx context.Context, appID string, email string, passwordHash string) error
	SaveEmailCode(ctx context.Context, appID string, email string, purpose string, codeHash string, expiresAt time.Time) error
	VerifyEmailCode(ctx context.Context, appID string, email string, purpose string, codeHash string, verifyTokenHash string, verifyTokenExpiresAt time.Time) error
	ConsumeVerifiedEmailCode(ctx context.Context, appID string, email string, purpose string, verifyTokenHash string) error
	ConsumeEmailCode(ctx context.Context, appID string, email string, purpose string, codeHash string) error
	CreateSession(ctx context.Context, userID string, expiresAt time.Time) (string, error)
	GetUserBySession(ctx context.Context, token string) (models.User, error)
	GetUser(ctx context.Context, userID string) (models.User, error)
	UpdateUserProfile(ctx context.Context, userID string, request models.UserProfileUpdateRequest) (models.User, error)
	DeleteUser(ctx context.Context, userID string) error
	AddCredits(ctx context.Context, userID string, delta int, reason string, refType string, refID string) (int, error)
	ConsumeCredits(ctx context.Context, userID string, amount int, reason string, refType string, refID string) (int, error)
	CreateCreditOrder(ctx context.Context, userID string, pkg models.CreditPackage) (models.CreditOrder, error)
	ListCreditPackages(ctx context.Context) ([]models.CreditPackage, error)
	RedeemCreditCode(ctx context.Context, userID string, cardNo string, password string) (int, error)
	ListImageModels(ctx context.Context) ([]models.ImageModel, error)
	GetImageModel(ctx context.Context, modelID string) (models.ImageModel, error)
	CreateGalleryImages(ctx context.Context, userID string, prompt string, style string, modelID string, modelName string, size string, quality string, creditsCost int, images []models.WebGeneratedImage) ([]models.WebGalleryImage, error)
	ListGalleryImages(ctx context.Context, userID string, limit int, before string, query string) ([]models.WebGalleryImage, error)
	ListFavoriteGalleryImages(ctx context.Context, userID string, limit int, before string, query string) ([]models.WebGalleryImage, error)
	SetGalleryImageLike(ctx context.Context, userID string, id string, liked bool) (models.WebGalleryImage, error)
	SetGalleryImageFavorite(ctx context.Context, userID string, id string, favorited bool) (models.WebGalleryImage, error)
	SetGalleryImageFeatured(ctx context.Context, userID string, id string, featured bool, promptFeatured bool) (models.WebGalleryImage, error)
	HasActiveWebImageTask(ctx context.Context, userID string) (bool, error)
	CreateWebImageTask(ctx context.Context, userID string, prompt string, style string, modelID string, size string, quality string, n int, creditsCost int) (models.WebImageTask, error)
	ListWebImageTasks(ctx context.Context, userID string, limit int) ([]models.WebImageTask, error)
	GetWebImageTask(ctx context.Context, userID string, id string) (models.WebImageTask, error)
	MarkWebImageTaskRunning(ctx context.Context, userID string, id string) error
	CompleteWebImageTask(ctx context.Context, userID string, id string, result models.WebGeneratedImage, galleryID string) (models.WebImageTask, error)
	FailWebImageTask(ctx context.Context, userID string, id string, errorMessage string) (models.WebImageTask, error)
	SetWebImageTaskPublic(ctx context.Context, userID string, id string, isPublic bool) (models.WebImageTask, error)
}

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (p *Postgres) CreateEmailUser(ctx context.Context, appID string, email string, passwordHash string) (models.User, error) {
	email = normalizeEmail(email)
	var userID string
	err := p.pool.QueryRow(ctx, `
		insert into users (app_id, email, email_normalized, password_hash, display_name)
		values ($1, $2, $2, $3, $2)
		on conflict (app_id, email_normalized) do nothing
		returning id::text
	`, appID, email, passwordHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, ErrConflict
	}
	if err != nil {
		return models.User{}, err
	}
	return p.GetUser(ctx, userID)
}

func (p *Postgres) GetEmailUser(ctx context.Context, appID string, email string) (models.User, error) {
	return p.scanUser(ctx, `
		select u.id::text, u.app_id, u.email, u.display_name, u.avatar_url, u.signature, u.gender,
			coalesce(a.balance, 0), coalesce(a.total_recharged, 0), u.created_at
		from users u
		left join user_credit_accounts a on a.user_id = u.id
		where u.app_id = $1 and u.email_normalized = $2
	`, appID, normalizeEmail(email))
}

func (p *Postgres) GetEmailPasswordHash(ctx context.Context, appID string, email string) (string, error) {
	var passwordHash string
	err := p.pool.QueryRow(ctx, `
		select password_hash from users where app_id = $1 and email_normalized = $2
	`, appID, normalizeEmail(email)).Scan(&passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return passwordHash, err
}

func (p *Postgres) SetEmailUserPassword(ctx context.Context, appID string, email string, passwordHash string) error {
	tag, err := p.pool.Exec(ctx, `
		update users set password_hash = $3, updated_at = now()
		where app_id = $1 and email_normalized = $2
	`, appID, normalizeEmail(email), passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) SaveEmailCode(ctx context.Context, appID string, email string, purpose string, codeHash string, expiresAt time.Time) error {
	_, err := p.pool.Exec(ctx, `
		update email_auth_codes
		set consumed_at = now()
		where email = $1 and purpose = $2 and consumed_at is null
	`, normalizeEmail(email), purpose)
	if err != nil {
		return err
	}
	_, err = p.pool.Exec(ctx, `
		insert into email_auth_codes (app_id, email, purpose, code_hash, expires_at)
		values ($1, $2, $3, $4, $5)
	`, appID, normalizeEmail(email), purpose, codeHash, expiresAt)
	return err
}

func (p *Postgres) VerifyEmailCode(ctx context.Context, appID string, email string, purpose string, codeHash string, verifyTokenHash string, verifyTokenExpiresAt time.Time) error {
	_ = appID
	tag, err := p.pool.Exec(ctx, `
		update email_auth_codes
		set verified_at = now(), verify_token_hash = $4, verify_token_expires_at = $5
		where id = (
			select id from email_auth_codes
			where email = $1 and purpose = $2 and code_hash = $3
				and consumed_at is null and expires_at > now()
			order by created_at desc
			limit 1
		)
	`, normalizeEmail(email), purpose, codeHash, verifyTokenHash, verifyTokenExpiresAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) ConsumeVerifiedEmailCode(ctx context.Context, appID string, email string, purpose string, verifyTokenHash string) error {
	_ = appID
	tag, err := p.pool.Exec(ctx, `
		update email_auth_codes
		set consumed_at = now()
		where id = (
			select id from email_auth_codes
			where email = $1 and purpose = $2 and verify_token_hash = $3
				and verified_at is not null and consumed_at is null and verify_token_expires_at > now()
			order by verified_at desc
			limit 1
		)
	`, normalizeEmail(email), purpose, verifyTokenHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) ConsumeEmailCode(ctx context.Context, appID string, email string, purpose string, codeHash string) error {
	_ = appID
	tag, err := p.pool.Exec(ctx, `
		update email_auth_codes
		set consumed_at = now()
		where id = (
			select id from email_auth_codes
			where email = $1 and purpose = $2 and code_hash = $3
				and consumed_at is null and expires_at > now()
			order by created_at desc
			limit 1
		)
	`, normalizeEmail(email), purpose, codeHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) CreateSession(ctx context.Context, userID string, expiresAt time.Time) (string, error) {
	token, err := randomToken(32)
	if err != nil {
		return "", err
	}
	_, err = p.pool.Exec(ctx, `
		insert into auth_sessions (token, user_id, app_id, expires_at)
		select $1, id, app_id, $3 from users where id = $2::uuid
	`, token, userID, expiresAt)
	return token, err
}

func (p *Postgres) GetUserBySession(ctx context.Context, token string) (models.User, error) {
	return p.scanUser(ctx, `
		select u.id::text, u.app_id, u.email, u.display_name, u.avatar_url, u.signature, u.gender,
			coalesce(a.balance, 0), coalesce(a.total_recharged, 0), u.created_at
		from auth_sessions s
		join users u on u.id = s.user_id
		left join user_credit_accounts a on a.user_id = u.id
		where s.token = $1 and s.expires_at > now()
	`, token)
}

func (p *Postgres) GetUser(ctx context.Context, userID string) (models.User, error) {
	return p.scanUser(ctx, `
		select u.id::text, u.app_id, u.email, u.display_name, u.avatar_url, u.signature, u.gender,
			coalesce(a.balance, 0), coalesce(a.total_recharged, 0), u.created_at
		from users u
		left join user_credit_accounts a on a.user_id = u.id
		where u.id = $1::uuid
	`, userID)
}

func (p *Postgres) UpdateUserProfile(ctx context.Context, userID string, request models.UserProfileUpdateRequest) (models.User, error) {
	tag, err := p.pool.Exec(ctx, `
		update users
		set display_name = case when $2 <> '' then $2 else display_name end,
			avatar_url = $3,
			signature = $4,
			gender = $5,
			updated_at = now()
		where id = $1::uuid
	`, userID, strings.TrimSpace(request.DisplayName), strings.TrimSpace(request.AvatarURL), strings.TrimSpace(request.Signature), strings.TrimSpace(request.Gender))
	if err != nil {
		return models.User{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.User{}, ErrNotFound
	}
	return p.GetUser(ctx, userID)
}

func (p *Postgres) DeleteUser(ctx context.Context, userID string) error {
	tag, err := p.pool.Exec(ctx, `delete from users where id = $1::uuid`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) AddCredits(ctx context.Context, userID string, delta int, reason string, refType string, refID string) (int, error) {
	if delta == 0 {
		user, err := p.GetUser(ctx, userID)
		return user.Credits, err
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var balance int
	err = tx.QueryRow(ctx, `
		insert into user_credit_accounts (user_id, balance, total_recharged)
		values ($1::uuid, $2, greatest($2, 0))
		on conflict (user_id) do update set
			balance = user_credit_accounts.balance + excluded.balance,
			total_recharged = user_credit_accounts.total_recharged + greatest(excluded.balance, 0),
			updated_at = now()
		returning balance
	`, userID, delta).Scan(&balance)
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `
		insert into credit_ledger (user_id, delta, balance_after, reason, ref_type, ref_id)
		values ($1::uuid, $2, $3, $4, $5, $6)
	`, userID, delta, balance, strings.TrimSpace(reason), strings.TrimSpace(refType), strings.TrimSpace(refID)); err != nil {
		return 0, err
	}
	return balance, tx.Commit(ctx)
}

func (p *Postgres) ConsumeCredits(ctx context.Context, userID string, amount int, reason string, refType string, refID string) (int, error) {
	if amount <= 0 {
		user, err := p.GetUser(ctx, userID)
		return user.Credits, err
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var balance int
	err = tx.QueryRow(ctx, `
		update user_credit_accounts
		set balance = balance - $2, updated_at = now()
		where user_id = $1::uuid and balance >= $2
		returning balance
	`, userID, amount).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrInsufficientCredits
	}
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `
		insert into credit_ledger (user_id, delta, balance_after, reason, ref_type, ref_id)
		values ($1::uuid, $2, $3, $4, $5, $6)
	`, userID, -amount, balance, strings.TrimSpace(reason), strings.TrimSpace(refType), strings.TrimSpace(refID)); err != nil {
		return 0, err
	}
	return balance, tx.Commit(ctx)
}

func (p *Postgres) CreateCreditOrder(ctx context.Context, userID string, pkg models.CreditPackage) (models.CreditOrder, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return models.CreditOrder{}, err
	}
	defer tx.Rollback(ctx)
	var order models.CreditOrder
	err = tx.QueryRow(ctx, `
		insert into credit_orders (user_id, package_id, credits, amount_cents, currency, status, provider, paid_at)
		values ($1::uuid, $2, $3, $4, $5, 'paid', 'manual', now())
		returning id::text, user_id::text, package_id, credits, amount_cents, currency, status, provider, created_at, paid_at
	`, userID, pkg.ID, pkg.Credits, pkg.AmountCents, pkg.Currency).Scan(
		&order.ID, &order.UserID, &order.PackageID, &order.Credits, &order.AmountCents, &order.Currency, &order.Status, &order.Provider, &order.CreatedAt, &order.PaidAt,
	)
	if err != nil {
		return models.CreditOrder{}, err
	}
	var balance int
	if err := tx.QueryRow(ctx, `
		insert into user_credit_accounts (user_id, balance, total_recharged)
		values ($1::uuid, $2, $2)
		on conflict (user_id) do update set
			balance = user_credit_accounts.balance + excluded.balance,
			total_recharged = user_credit_accounts.total_recharged + excluded.balance,
			updated_at = now()
		returning balance
	`, userID, pkg.Credits).Scan(&balance); err != nil {
		return models.CreditOrder{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into credit_ledger (user_id, delta, balance_after, reason, ref_type, ref_id)
		values ($1::uuid, $2, $3, 'purchase', 'credit_order', $4)
	`, userID, pkg.Credits, balance, order.ID); err != nil {
		return models.CreditOrder{}, err
	}
	return order, tx.Commit(ctx)
}

func (p *Postgres) ListCreditPackages(ctx context.Context) ([]models.CreditPackage, error) {
	rows, err := p.pool.Query(ctx, `
		select package_id, name, credits, amount_cents, currency, icon, payment_url
		from credit_package_configs
		where enabled = true
		order by sort_order, credits
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.CreditPackage
	for rows.Next() {
		var item models.CreditPackage
		if err := rows.Scan(&item.ID, &item.Name, &item.Credits, &item.AmountCents, &item.Currency, &item.Icon, &item.PaymentURL); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *Postgres) RedeemCreditCode(ctx context.Context, userID string, cardNo string, password string) (int, error) {
	cardNo = strings.TrimSpace(cardNo)
	password = strings.TrimSpace(password)
	if cardNo == "" || password == "" {
		return 0, ErrNotFound
	}
	passwordHash := creditRedeemPasswordHash(cardNo, password)
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var credits int
	err = tx.QueryRow(ctx, `
		update credit_redeem_codes
		set status = 'redeemed', redeemed_by = $2::uuid, redeemed_at = now()
		where lower(code) = lower($1) and password_hash = $3 and status = 'unused'
		returning credits
	`, cardNo, userID, passwordHash).Scan(&credits)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	var balance int
	if err := tx.QueryRow(ctx, `
		insert into user_credit_accounts (user_id, balance, total_recharged)
		values ($1::uuid, $2, $2)
		on conflict (user_id) do update set
			balance = user_credit_accounts.balance + excluded.balance,
			total_recharged = user_credit_accounts.total_recharged + excluded.balance,
			updated_at = now()
		returning balance
	`, userID, credits).Scan(&balance); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `
		insert into credit_ledger (user_id, delta, balance_after, reason, ref_type, ref_id)
		values ($1::uuid, $2, $3, 'redeem_code', 'credit_redeem_code', $4)
	`, userID, credits, balance, cardNo); err != nil {
		return 0, err
	}
	return credits, tx.Commit(ctx)
}

func creditRedeemPasswordHash(cardNo string, password string) string {
	sum := sha256.Sum256([]byte(strings.ToUpper(strings.TrimSpace(cardNo)) + "|" + strings.TrimSpace(password)))
	return hex.EncodeToString(sum[:])
}

func (p *Postgres) ListImageModels(ctx context.Context) ([]models.ImageModel, error) {
	rows, err := p.pool.Query(ctx, `
		select id, name, provider, description, credit_cost, enabled
		from image_models
		where enabled = true
		order by sort_order, credit_cost, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.ImageModel
	for rows.Next() {
		var item models.ImageModel
		if err := rows.Scan(&item.ID, &item.Name, &item.Provider, &item.Description, &item.CreditCost, &item.Enabled); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *Postgres) GetImageModel(ctx context.Context, modelID string) (models.ImageModel, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		modelID = "gpt-image"
	}
	var item models.ImageModel
	err := p.pool.QueryRow(ctx, `
		select id, name, provider, description, credit_cost, enabled
		from image_models
		where id = $1 and enabled = true
	`, modelID).Scan(&item.ID, &item.Name, &item.Provider, &item.Description, &item.CreditCost, &item.Enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ImageModel{}, ErrNotFound
	}
	return item, err
}

func (p *Postgres) CreateGalleryImages(ctx context.Context, userID string, prompt string, style string, modelID string, modelName string, size string, quality string, creditsCost int, images []models.WebGeneratedImage) ([]models.WebGalleryImage, error) {
	if len(images) == 0 {
		return nil, nil
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	items := make([]models.WebGalleryImage, 0, len(images))
	for _, image := range images {
		row := tx.QueryRow(ctx, `
			insert into web_gallery_images (user_id, prompt, style, model_id, model_name, image_data, mime_type, size, quality, credits_cost)
			values (nullif($1, '')::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			returning id::text, coalesce(user_id::text, ''),
				coalesce((select display_name from users where id = nullif($1, '')::uuid), ''),
				coalesce((select avatar_url from users where id = nullif($1, '')::uuid), ''),
				image_data, prompt, style,
				coalesce(model_id, ''), coalesce(model_name, ''), mime_type, size, quality, credits_cost,
				is_public, is_featured, is_prompt_featured, 0, false, 0, false, created_at
		`, strings.TrimSpace(userID), prompt, style, modelID, modelName, image.URL, firstNonEmptyStore(image.MimeType, "image/png"), size, quality, creditsCost)
		item, err := scanGalleryImage(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, tx.Commit(ctx)
}

func (p *Postgres) ListGalleryImages(ctx context.Context, userID string, limit int, before string, query string) ([]models.WebGalleryImage, error) {
	return p.listGalleryImages(ctx, userID, limit, before, false, query)
}

func (p *Postgres) ListFavoriteGalleryImages(ctx context.Context, userID string, limit int, before string, query string) ([]models.WebGalleryImage, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrNotFound
	}
	return p.listGalleryImages(ctx, userID, limit, before, true, query)
}

func (p *Postgres) listGalleryImages(ctx context.Context, userID string, limit int, before string, favoritesOnly bool, query string) ([]models.WebGalleryImage, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	args := []any{limit, strings.TrimSpace(userID)}
	userParam := "$2"
	cursorFilter := ""
	if strings.TrimSpace(before) != "" {
		args = []any{limit, strings.TrimSpace(before), strings.TrimSpace(userID)}
		userParam = "$3"
		cursorFilter = "and g.created_at < coalesce((select created_at from web_gallery_images where id::text = $2 limit 1), 'infinity'::timestamptz)"
	}
	favoriteFilter := ""
	if favoritesOnly {
		favoriteFilter = "and exists (select 1 from web_gallery_image_favorites fav where fav.image_id = g.id and fav.user_id = " + userParam + "::uuid)"
	}
	searchFilter := ""
	if strings.TrimSpace(query) != "" {
		args = append(args, "%"+strings.TrimSpace(query)+"%")
		searchFilter = "and (g.prompt ilike $" + strconv.Itoa(len(args)) + " or g.style ilike $" + strconv.Itoa(len(args)) + " or g.model_name ilike $" + strconv.Itoa(len(args)) + ")"
	}
	rows, err := p.pool.Query(ctx, `
		select g.id::text, coalesce(g.user_id::text, ''),
			coalesce(u.display_name, ''), coalesce(u.avatar_url, ''),
			g.image_data, g.prompt, g.style,
			coalesce(g.model_id, ''), coalesce(g.model_name, ''), g.mime_type, g.size, g.quality, g.credits_cost,
			g.is_public, g.is_featured, g.is_prompt_featured,
			coalesce(like_counts.like_count, 0),
			case when nullif(`+userParam+`, '') is null then false else exists (
				select 1 from web_gallery_image_likes likes
				where likes.image_id = g.id and likes.user_id = `+userParam+`::uuid
			) end as liked_by_me,
			coalesce(favorite_counts.favorite_count, 0),
			case when nullif(`+userParam+`, '') is null then false else exists (
				select 1 from web_gallery_image_favorites favorites
				where favorites.image_id = g.id and favorites.user_id = `+userParam+`::uuid
			) end as favorited_by_me,
			g.created_at
		from web_gallery_images g
		left join users u on u.id = g.user_id
		left join (
			select image_id, count(*)::int as like_count
			from web_gallery_image_likes
			group by image_id
		) like_counts on like_counts.image_id = g.id
		left join (
			select image_id, count(*)::int as favorite_count
			from web_gallery_image_favorites
			group by image_id
		) favorite_counts on favorite_counts.image_id = g.id
		where g.is_public = true `+cursorFilter+` `+favoriteFilter+` `+searchFilter+`
		order by g.is_featured desc, g.created_at desc
		limit $1
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.WebGalleryImage
	for rows.Next() {
		item, err := scanGalleryImage(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *Postgres) SetGalleryImageLike(ctx context.Context, userID string, id string, liked bool) (models.WebGalleryImage, error) {
	if liked {
		_, err := p.pool.Exec(ctx, `
			insert into web_gallery_image_likes (image_id, user_id)
			values ($1::uuid, $2::uuid)
			on conflict (image_id, user_id) do nothing
		`, id, userID)
		if err != nil {
			return models.WebGalleryImage{}, err
		}
	} else {
		if _, err := p.pool.Exec(ctx, `delete from web_gallery_image_likes where image_id = $1::uuid and user_id = $2::uuid`, id, userID); err != nil {
			return models.WebGalleryImage{}, err
		}
	}
	return p.getGalleryImage(ctx, userID, id)
}

func (p *Postgres) SetGalleryImageFavorite(ctx context.Context, userID string, id string, favorited bool) (models.WebGalleryImage, error) {
	if favorited {
		_, err := p.pool.Exec(ctx, `
			insert into web_gallery_image_favorites (image_id, user_id)
			values ($1::uuid, $2::uuid)
			on conflict (image_id, user_id) do nothing
		`, id, userID)
		if err != nil {
			return models.WebGalleryImage{}, err
		}
	} else {
		if _, err := p.pool.Exec(ctx, `delete from web_gallery_image_favorites where image_id = $1::uuid and user_id = $2::uuid`, id, userID); err != nil {
			return models.WebGalleryImage{}, err
		}
	}
	return p.getGalleryImage(ctx, userID, id)
}

func (p *Postgres) SetGalleryImageFeatured(ctx context.Context, userID string, id string, featured bool, promptFeatured bool) (models.WebGalleryImage, error) {
	tag, err := p.pool.Exec(ctx, `
		update web_gallery_images
		set is_featured = $3, is_prompt_featured = $4
		where id = $1::uuid and user_id = $2::uuid
	`, id, userID, featured, promptFeatured)
	if err != nil {
		return models.WebGalleryImage{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.WebGalleryImage{}, ErrNotFound
	}
	return p.getGalleryImage(ctx, userID, id)
}

func (p *Postgres) CreateWebImageTask(ctx context.Context, userID string, prompt string, style string, modelID string, size string, quality string, n int, creditsCost int) (models.WebImageTask, error) {
	row := p.pool.QueryRow(ctx, webImageTaskSelect(`
		insert into web_image_tasks (user_id, prompt, style, model_id, model_name, size, quality, n, credits_cost)
		values ($1::uuid, $2, $3, $4, coalesce((select name from image_models where id = $4), ''), $5, $6, $7, $8)
		returning
	`), userID, prompt, style, strings.TrimSpace(modelID), size, quality, n, creditsCost)
	task, err := scanWebImageTask(row)
	if err != nil && strings.Contains(err.Error(), "web_image_tasks_one_active_per_user_idx") {
		return models.WebImageTask{}, ErrConflict
	}
	return task, err
}

func (p *Postgres) HasActiveWebImageTask(ctx context.Context, userID string) (bool, error) {
	var active bool
	err := p.pool.QueryRow(ctx, `
		select exists (
			select 1
			from web_image_tasks
			where user_id = $1::uuid and status in ('queued', 'running')
		)
	`, userID).Scan(&active)
	return active, err
}

func (p *Postgres) ListWebImageTasks(ctx context.Context, userID string, limit int) ([]models.WebImageTask, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 60 {
		limit = 60
	}
	rows, err := p.pool.Query(ctx, `
		select id::text, user_id::text, prompt, style, coalesce(model_id, ''), coalesce(model_name, ''), size, quality, n, credits_cost,
			status, error_message, result_image_data, result_mime_type,
			coalesce(gallery_image_id::text, ''), is_public, created_at, started_at, completed_at, updated_at
		from web_image_tasks
		where user_id = $1::uuid
		order by created_at desc
		limit $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.WebImageTask
	for rows.Next() {
		item, err := scanWebImageTask(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *Postgres) GetWebImageTask(ctx context.Context, userID string, id string) (models.WebImageTask, error) {
	row := p.pool.QueryRow(ctx, webImageTaskSelect(`
		select
	`)+` from web_image_tasks where id = $1::uuid and user_id = $2::uuid`, id, userID)
	task, err := scanWebImageTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.WebImageTask{}, ErrNotFound
	}
	return task, err
}

func (p *Postgres) MarkWebImageTaskRunning(ctx context.Context, userID string, id string) error {
	tag, err := p.pool.Exec(ctx, `
		update web_image_tasks
		set status = 'running', started_at = coalesce(started_at, now()), updated_at = now()
		where id = $1::uuid and user_id = $2::uuid and status = 'queued'
	`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) CompleteWebImageTask(ctx context.Context, userID string, id string, result models.WebGeneratedImage, galleryID string) (models.WebImageTask, error) {
	row := p.pool.QueryRow(ctx, webImageTaskSelect(`
		update web_image_tasks
		set status = 'succeeded',
			error_message = '',
			result_image_data = case when nullif($5, '') is null then $3 else '' end,
			result_mime_type = case when nullif($5, '') is null then $4 else '' end,
			gallery_image_id = nullif($5, '')::uuid,
			completed_at = now(),
			updated_at = now()
		where id = $1::uuid and user_id = $2::uuid
		returning
	`), id, userID, result.URL, firstNonEmptyStore(result.MimeType, "image/png"), strings.TrimSpace(galleryID))
	task, err := scanWebImageTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.WebImageTask{}, ErrNotFound
	}
	return task, err
}

func (p *Postgres) FailWebImageTask(ctx context.Context, userID string, id string, errorMessage string) (models.WebImageTask, error) {
	row := p.pool.QueryRow(ctx, webImageTaskSelect(`
		update web_image_tasks
		set status = 'failed', error_message = $3, completed_at = now(), updated_at = now()
		where id = $1::uuid and user_id = $2::uuid
		returning
	`), id, userID, strings.TrimSpace(errorMessage))
	task, err := scanWebImageTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.WebImageTask{}, ErrNotFound
	}
	return task, err
}

func (p *Postgres) SetWebImageTaskPublic(ctx context.Context, userID string, id string, isPublic bool) (models.WebImageTask, error) {
	row := p.pool.QueryRow(ctx, webImageTaskSelect(`
		update web_image_tasks
		set is_public = $3, updated_at = now()
		where id = $1::uuid and user_id = $2::uuid
		returning
	`), id, userID, isPublic)
	task, err := scanWebImageTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.WebImageTask{}, ErrNotFound
	}
	return task, err
}

func (p *Postgres) scanUser(ctx context.Context, query string, args ...any) (models.User, error) {
	var user models.User
	err := p.pool.QueryRow(ctx, query, args...).Scan(
		&user.ID,
		&user.AppID,
		&user.Email,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Signature,
		&user.Gender,
		&user.Credits,
		&user.TotalRecharged,
		&user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	return user, err
}

func (p *Postgres) getGalleryImage(ctx context.Context, userID string, id string) (models.WebGalleryImage, error) {
	row := p.pool.QueryRow(ctx, `
		select g.id::text, coalesce(g.user_id::text, ''),
			coalesce(u.display_name, ''), coalesce(u.avatar_url, ''),
			g.image_data, g.prompt, g.style,
			coalesce(g.model_id, ''), coalesce(g.model_name, ''), g.mime_type, g.size, g.quality, g.credits_cost,
			g.is_public, g.is_featured, g.is_prompt_featured,
			(select count(*)::int from web_gallery_image_likes where image_id = g.id),
			case when nullif($2, '') is null then false else exists (
				select 1 from web_gallery_image_likes where image_id = g.id and user_id = $2::uuid
			) end,
			(select count(*)::int from web_gallery_image_favorites where image_id = g.id),
			case when nullif($2, '') is null then false else exists (
				select 1 from web_gallery_image_favorites where image_id = g.id and user_id = $2::uuid
			) end,
			g.created_at
		from web_gallery_images g
		left join users u on u.id = g.user_id
		where g.id = $1::uuid
	`, id, strings.TrimSpace(userID))
	item, err := scanGalleryImage(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.WebGalleryImage{}, ErrNotFound
	}
	return item, err
}

func scanGalleryImage(row interface{ Scan(dest ...any) error }) (models.WebGalleryImage, error) {
	var item models.WebGalleryImage
	var mimeType string
	var createdAt time.Time
	err := row.Scan(
		&item.ID,
		&item.UserID,
		&item.Author,
		&item.AuthorAvatarURL,
		&item.Image,
		&item.Prompt,
		&item.Style,
		&item.ModelID,
		&item.ModelName,
		&mimeType,
		&item.Size,
		&item.Quality,
		&item.CreditsCost,
		&item.IsPublic,
		&item.IsFeatured,
		&item.IsPromptFeatured,
		&item.LikeCount,
		&item.LikedByMe,
		&item.FavoriteCount,
		&item.FavoritedByMe,
		&createdAt,
	)
	if err != nil {
		return models.WebGalleryImage{}, err
	}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.Ratio = webGalleryRatio(item.Size)
	item.Tag = firstNonEmptyStore(item.Style, "新作品")
	return item, nil
}

func webImageTaskSelect(prefix string) string {
	return prefix + `
		id::text, user_id::text, prompt, style, coalesce(model_id, ''), coalesce(model_name, ''), size, quality, n, credits_cost,
		status, error_message, result_image_data, result_mime_type,
		coalesce(gallery_image_id::text, ''), is_public, created_at, started_at, completed_at, updated_at
	`
}

func scanWebImageTask(row interface{ Scan(dest ...any) error }) (models.WebImageTask, error) {
	var task models.WebImageTask
	var createdAt, updatedAt time.Time
	var startedAt, completedAt sql.NullTime
	err := row.Scan(
		&task.ID,
		&task.UserID,
		&task.Prompt,
		&task.Style,
		&task.ModelID,
		&task.ModelName,
		&task.Size,
		&task.Quality,
		&task.N,
		&task.CreditsCost,
		&task.Status,
		&task.ErrorMessage,
		&task.ResultImage,
		&task.ResultMimeType,
		&task.GalleryImageID,
		&task.IsPublic,
		&createdAt,
		&startedAt,
		&completedAt,
		&updatedAt,
	)
	if err != nil {
		return models.WebImageTask{}, err
	}
	task.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	task.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	if startedAt.Valid {
		task.StartedAt = startedAt.Time.UTC().Format(time.RFC3339)
	}
	if completedAt.Valid {
		task.CompletedAt = completedAt.Time.UTC().Format(time.RFC3339)
	}
	return task, nil
}

func webGalleryRatio(size string) string {
	parts := strings.Split(strings.TrimSpace(size), "x")
	if len(parts) != 2 {
		return "tall"
	}
	width, _ := strconv.Atoi(parts[0])
	height, _ := strconv.Atoi(parts[1])
	if width > height {
		return "wide"
	}
	if width == height {
		return "square"
	}
	return "tall"
}

func randomToken(size int) (string, error) {
	token := make([]byte, size)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return hex.EncodeToString(token), nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func firstNonEmptyStore(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
