create or replace function record_invalid_captcha(ip inet)
returns int
language sql strict
as $$
	insert into failed_captchas (ip, expires)
		values (record_invalid_captcha.ip, now() + interval '1 hour');
	select count(*)
		from failed_captchas f
		where ip = record_invalid_captcha.ip and expires > now()
$$;
