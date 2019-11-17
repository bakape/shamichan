create or replace function validate_captcha(session bytea)
returns void
language sql strict
as $$
	insert into last_solved_captchas (token)
		values (validate_captcha.session)
		on conflict (token) do
		update set time = now();
	delete from spam_scores
		where token = validate_captcha.session;
$$;
