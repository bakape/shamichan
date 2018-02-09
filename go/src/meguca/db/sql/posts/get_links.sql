select l.source, l.target, p.op, t.board
from links as l
join posts as p on l.target = p.id
join threads as t on p.op = t.id
where l.source = any($1::bigint[])
