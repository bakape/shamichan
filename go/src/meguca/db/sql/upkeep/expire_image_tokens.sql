delete from image_tokens
	where expires < now() at time zone 'utc'
