select to_jsonb(m)
from mod_log m
where m.id = $1
