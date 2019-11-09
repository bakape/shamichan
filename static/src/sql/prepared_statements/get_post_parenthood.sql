select board, op
from posts p
join threads t on t.id = p.op
where p.id = $1
