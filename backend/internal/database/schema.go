package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func EnsureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	statements := []string{
		`create extension if not exists pgcrypto`,
		`create table if not exists users (
			id uuid primary key default gen_random_uuid(),
			app_id text not null default 'berserk.web',
			email text not null default '',
			email_normalized text not null default '',
			password_hash text not null default '',
			display_name text not null default '',
			avatar_url text not null default '',
			signature text not null default '',
			gender text not null default '',
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			unique (app_id, email_normalized)
		)`,
		`create table if not exists auth_sessions (
			token text primary key,
			user_id uuid not null references users(id) on delete cascade,
			app_id text not null default 'berserk.web',
			expires_at timestamptz not null,
			created_at timestamptz not null default now()
		)`,
		`create index if not exists auth_sessions_user_idx on auth_sessions (user_id)`,
		`create table if not exists email_auth_codes (
			id uuid primary key default gen_random_uuid(),
			app_id text not null default 'berserk.web',
			email text not null,
			purpose text not null,
			code_hash text not null,
			attempts integer not null default 0,
			expires_at timestamptz not null,
			verified_at timestamptz,
			verify_token_hash text not null default '',
			verify_token_expires_at timestamptz,
			consumed_at timestamptz,
			created_at timestamptz not null default now()
		)`,
		`create index if not exists email_auth_codes_lookup_idx on email_auth_codes (app_id, email, purpose, created_at desc)`,
		`create table if not exists user_credit_accounts (
			user_id uuid primary key references users(id) on delete cascade,
			balance integer not null default 0,
			total_recharged integer not null default 0,
			updated_at timestamptz not null default now()
		)`,
		`create table if not exists credit_ledger (
			id uuid primary key default gen_random_uuid(),
			user_id uuid not null references users(id) on delete cascade,
			delta integer not null,
			balance_after integer not null,
			reason text not null default '',
			ref_type text not null default '',
			ref_id text not null default '',
			created_at timestamptz not null default now()
		)`,
		`create index if not exists credit_ledger_user_created_idx on credit_ledger (user_id, created_at desc)`,
		`create table if not exists credit_orders (
			id uuid primary key default gen_random_uuid(),
			user_id uuid not null references users(id) on delete cascade,
			package_id text not null,
			credits integer not null,
			amount_cents integer not null,
			currency text not null default 'CNY',
			status text not null default 'pending',
			provider text not null default 'card_key',
			created_at timestamptz not null default now(),
			paid_at timestamptz
		)`,
		`create table if not exists credit_package_configs (
			package_id text primary key,
			name text not null,
			credits integer not null,
			amount_cents integer not null,
			currency text not null default 'CNY',
			icon text not null default '',
			payment_url text not null default '',
			enabled boolean not null default true,
			sort_order integer not null default 100,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now()
		)`,
		`insert into credit_package_configs (package_id, name, credits, amount_cents, currency, icon, payment_url, enabled, sort_order)
		values
			('credits_trial', '限时体验包', 10, 100, 'CNY', '/pricing-icons/package-trial.png', '', true, 5),
			('credits_100', '灵感入门包', 110, 1000, 'CNY', '/pricing-icons/package-100.png', '', true, 10),
			('credits_500', '创作加速包', 550, 4900, 'CNY', '/pricing-icons/package-500.png', '', true, 20),
			('credits_1000', '高频创作包', 1100, 9500, 'CNY', '/pricing-icons/package-1000.png', '', true, 30),
			('credits_5000', '工作室储备包', 5000, 45000, 'CNY', '/pricing-icons/credits-5000.png', '', false, 40)
		on conflict (package_id) do update set
			name = excluded.name,
			credits = excluded.credits,
			amount_cents = excluded.amount_cents,
			currency = excluded.currency,
			icon = excluded.icon,
			enabled = excluded.enabled,
			sort_order = excluded.sort_order,
			updated_at = now()`,
		`create table if not exists credit_redeem_codes (
			code text primary key,
			package_id text not null default '',
			credits integer not null,
			password_hash text not null default '',
			status text not null default 'unused',
			redeemed_by uuid references users(id) on delete set null,
			redeemed_at timestamptz,
			created_at timestamptz not null default now()
		)`,
		`alter table credit_redeem_codes add column if not exists password_hash text not null default ''`,
		`update credit_redeem_codes set credits = case package_id
			when 'credits_trial' then 10
			when 'credits_100' then 110
			when 'credits_500' then 550
			when 'credits_1000' then 1100
			else credits
		end
		where status = 'unused'
			and package_id in ('credits_trial', 'credits_100', 'credits_500', 'credits_1000')`,
		`create table if not exists image_models (
			id text primary key,
			name text not null,
			provider text not null default '',
			description text not null default '',
			credit_cost integer not null default 5,
			enabled boolean not null default true,
			sort_order integer not null default 100,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now()
		)`,
		`insert into image_models (id, name, provider, description, credit_cost, enabled, sort_order)
		values
			('gpt-image', 'GPT Image', 'OpenAI', '通用商业海报与插画', 5, true, 10),
			('seedream', 'Seedream', 'ByteDance', '中文提示词友好', 6, true, 20),
			('qwen-image', 'Qwen Image', 'Alibaba', '产品图与中文排版', 5, true, 30),
			('gemini-image', 'Gemini Image', 'Google', '多模态参考图创作', 8, true, 40)
		on conflict (id) do update set
			name = excluded.name,
			provider = excluded.provider,
			description = excluded.description,
			updated_at = now()`,
		`create table if not exists web_gallery_images (
			id uuid primary key default gen_random_uuid(),
			user_id uuid references users(id) on delete set null,
			prompt text not null default '',
			style text not null default '',
			model_id text not null default '',
			model_name text not null default '',
			image_data text not null default '',
			mime_type text not null default 'image/png',
			size text not null default '',
			quality text not null default '',
			credits_cost integer not null default 0,
			is_public boolean not null default true,
			is_featured boolean not null default false,
			is_prompt_featured boolean not null default false,
			created_at timestamptz not null default now()
		)`,
		`create index if not exists web_gallery_images_public_created_idx on web_gallery_images (is_public, is_featured, created_at desc)`,
		`create table if not exists web_gallery_image_likes (
			image_id uuid not null references web_gallery_images(id) on delete cascade,
			user_id uuid not null references users(id) on delete cascade,
			created_at timestamptz not null default now(),
			primary key (image_id, user_id)
		)`,
		`create index if not exists web_gallery_image_likes_image_idx on web_gallery_image_likes (image_id)`,
		`create table if not exists web_gallery_image_favorites (
			image_id uuid not null references web_gallery_images(id) on delete cascade,
			user_id uuid not null references users(id) on delete cascade,
			created_at timestamptz not null default now(),
			primary key (image_id, user_id)
		)`,
		`create index if not exists web_gallery_image_favorites_user_created_idx on web_gallery_image_favorites (user_id, created_at desc)`,
		`create index if not exists web_gallery_image_favorites_image_idx on web_gallery_image_favorites (image_id)`,
		`create table if not exists web_image_tasks (
			id uuid primary key default gen_random_uuid(),
			user_id uuid not null references users(id) on delete cascade,
			prompt text not null default '',
			style text not null default '',
			model_id text not null default '',
			model_name text not null default '',
			size text not null default '1024x1536',
			quality text not null default 'medium',
			n integer not null default 1,
			credits_cost integer not null default 0,
			status text not null default 'queued',
			error_message text not null default '',
			result_image_data text not null default '',
			result_mime_type text not null default '',
			gallery_image_id uuid references web_gallery_images(id) on delete set null,
			is_public boolean not null default true,
			created_at timestamptz not null default now(),
			started_at timestamptz,
			completed_at timestamptz,
			updated_at timestamptz not null default now()
		)`,
		`create unique index if not exists web_image_tasks_one_active_per_user_idx
			on web_image_tasks (user_id)
			where status in ('queued', 'running')`,
		`create index if not exists web_image_tasks_user_created_idx on web_image_tasks (user_id, created_at desc)`,
	}
	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}
