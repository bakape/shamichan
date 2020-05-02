-- Parent table for all expiring tables
create table expiries (
	expires timestamptz not null
);

-- For faster cleanup calls
create index expiries_expires_idx on expiries (expires);

-- Sliding antispam scores
create table spam_scores (
	public_key bigint primary key references public_keys
)
inherits (expiries);

-- Last solved captcha for user
create table last_solved_captchas (
	public_key bigint primary key references public_keys
)
inherits (expiries);
