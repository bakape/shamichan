select id, banned, deleted, spoiler from posts
where op = $1
order by id asc