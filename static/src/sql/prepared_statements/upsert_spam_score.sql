insert into spam_scores (token, score)
values ($1, $2)
on conflict (token)
do update set score = EXCLUDED.score
