with t as (
	select editing, banned, spoiler, deleted, sage, id, time, body, name, trip,
		auth, links, backlinks, commands, imageName, images.*
	from posts
	left outer join images
		on posts.SHA1 = images.SHA1
	where op = $1 and id != $1
	order by id desc
	limit $2
)
select * from t
	order by id asc
