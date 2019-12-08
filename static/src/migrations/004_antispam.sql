-- Parent table fro all expiring tables
create table expiries (
	expires timestamptz not null
);

-- For faster cleanup calls
create index expiries_expires_idx on expiries (expires);

-- Parent table fro all expiring tables with an auth_key column
create table auth_expiries (
	auth_key auth_key not null
)
inherits (expiries);

-- Sliding antispam scores
create table spam_scores (
	primary key (auth_key)
)
inherits (auth_expiries);

-- Last solved captcha for user
create table last_solved_captchas (
	primary key (auth_key)
)
inherits (auth_expiries);

-- Incorrectly submitted captchas
create table failed_captchas () inherits (auth_expiries);
