select readOnly, textOnly, forcedAnon, disableRobots, id,
	defaultCSS, title, notice, rules, eightball
	from boards
	where id = $1
