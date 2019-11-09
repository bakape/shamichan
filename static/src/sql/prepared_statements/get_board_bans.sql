select ip, forPost, reason, "by", expires, type
from bans
where expires > now() and board = $1
order by expires desc
