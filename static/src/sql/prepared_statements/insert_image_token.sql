insert into image_tokens (token, sha1, expires)
values ($1, $2, now() + interval '1 minute')
