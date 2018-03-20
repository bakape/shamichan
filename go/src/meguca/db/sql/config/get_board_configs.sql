select readOnly, textOnly, forcedAnon, disableRobots, flags, NSFW, nonLive,
		posterIDs, rbText,
		id,	defaultCSS, title, notice, rules, eightball
	from boards
	where id = $1
