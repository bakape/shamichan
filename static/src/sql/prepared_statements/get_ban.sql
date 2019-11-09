select ip, board, forPost, reason, "by", expires
from bans
where expires > now()
	and ip = $1
	and board in ('all', $2)
	and type = 'classic'
order by expires desc
limit 1
