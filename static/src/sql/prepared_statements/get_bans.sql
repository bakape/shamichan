select distinct ip, board
from bans
where expires > now() and type = 'classic'
order by ip, board, expires desc
