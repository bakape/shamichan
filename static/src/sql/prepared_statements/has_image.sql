select exists (
	select
	from posts
	where id = $1 and sha1 is not null
)
