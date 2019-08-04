select p.board, i.sha1, i.file_type, i.thumb_type
from posts p
left join images i on p.sha1 = i.sha1
where p.id = $1
