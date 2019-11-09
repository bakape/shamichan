select jsonb_agg(m order by m.created desc)
from mod_log m
where m.board = $1
