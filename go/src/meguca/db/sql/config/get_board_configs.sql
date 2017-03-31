select readOnly, textOnly, forcedAnon, id, title, notice, rules, eightball
	from boards
	where id = $1
