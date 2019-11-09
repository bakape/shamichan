-- Uses up an image allocation token and returns the image's ID
create function use_image_token(token char(86))
returns char(40)
language plpgsql strict
as $$
declare
	sha1 char(40);
begin
	delete from image_tokens
		where image_tokens.token = use_image_token.token
		returning image_tokens.sha1 into sha1;
	if sha1 is null then
		raise exception 'invalid image token';
	end if;
	return sha1;
end;
$$;
