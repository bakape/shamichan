select exists (
	select
	from images
	where sha1 = $1
)
