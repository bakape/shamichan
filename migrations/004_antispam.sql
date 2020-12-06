-- Parent table for all expiring tables
create table expiries (
	expires timestamptz not null
);

-- Sliding antispam scores
create table spam_scores (
	public_key bigint primary key references public_keys
)
inherits (expiries);
create index spam_scores_expires_idx on spam_scores (expires);

-- Last solved captcha for user
create table last_solved_captchas (
	public_key bigint primary key references public_keys
)
inherits (expiries);
create index last_solved_captchas_expires_idx on last_solved_captchas (expires);
