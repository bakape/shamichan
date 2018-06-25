// Link to a post
export type PostLink = {
	id: number
	op: number
	board: string
}
// Server-wide global configurations
export type Configs = {
	captcha: boolean
	mature: boolean // Website intended for mature audiences
	disableUserBoards: boolean
	pruneThreads: boolean
	threadExpiryMin: number
	threadExpiryMax: number
	maxSize: number
	defaultLang: string
	defaultCSS: string
	imageRootOverride: string
	links: { [key: string]: string }
}

export type LanguagePack = {
	posts: { [key: string]: string }
	plurals: { [key: string]: [string, string] }
	time: {
		calendar: string[]
		week: string[]
	}
	ui: { [key: string]: string }
	sync: string[]
}
