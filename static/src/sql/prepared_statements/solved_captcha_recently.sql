select exists (
	select
	from last_solved_captchas
	where token = $1 and time > $2
)
