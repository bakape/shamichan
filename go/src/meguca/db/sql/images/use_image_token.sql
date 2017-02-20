delete from image_tokens
	where token = $1
	returning SHA1
