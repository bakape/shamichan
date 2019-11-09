select i.sha1, i.file_type, i.length
from images i
where exists (
	select
	from posts as p
	where p.sha1 = i.sha1 and p.board = $1)
	and file_type in (3, 6)
	and audio = true
	and video = true
	and length between 10 and 600
)
order by random()
