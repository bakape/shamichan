// Format a board name and title into cannonical board header format
export const formatHeader = (name: string, title: string): string =>
	`/${name}/ - ${title}`
