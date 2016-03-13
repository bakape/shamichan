/*
 Provides type-safe and selective mappings for the language packs
*/

type LanguagePack = {
	posts: LnPosts
	banner: LnBanner
	images: LnImages
	navigation: LnNavigation
	reports: LnReports
	time: LnTime
	sync: LnSync
	syncwatch: LnSyncwatch
	opts: LnOpts
}

declare var lang: LanguagePack

// Export each container indivudually for namespacing purposes
// Can't use destructuring, because it breaks with the SystemJS module compiler.
export const posts = lang.posts
export const banner = lang.banner
export const images = lang.images
export const navigation = lang.navigation
export const reports = lang.reports
export const time = lang.time
export const sync = lang.sync
export const syncwatch = lang.syncwatch
export const opts = lang.opts

type LnPosts = {
	anon: string
	newThread: string
	reply: string
	you: string
	OP: string
	locked: string
	uploading: string
	subject: string
	received: string
	unknownUpload: string
	unknownResult: string
	threadLocked: string
	quoted: string
	cancel: string
}

type LnBanner = {
	showSeconds: string
	worksBestWith: string
	name: string
	email: string
	options: string
	identity: string
	faq: string
	feedback: string
	onlineCounter: string
	googleSong: string
}

type LnImages = {
	show: string
	hide: string
	expand: string
	contract: string
}

type LnNavigation = {
	search: string
	rescan: string
	report: string
	focus: string
	last: string
	bottom: string
	expand: string
	catalog: string
	return: string
	top: string
	lockedToBottom: string
	seeAll: string
	catalogOmit: string
}

type LnReports = {
	post: string
	reporting: string
	submitted: string
	setup: string
	leadError: string
}

type LnTime = {
	week: string[]
	calendar: string[]
	justNow: string
	minute: string
	minutes: string
	hour: string
	hours: string
	day: string
	days: string
	month: string
	months: string
	year: string
	years: string
}

type LnSync = {
	0: string
	1: string
	2: string
}

type LnSyncwatch = {
	starting: string
	finished: string
}

type LnOpts = {
	tabs: string[]
	modes: {[mode: string]: string}
	importConfig: {
		done: string
		corrupt: string
	}
	langApplied: string
	labels: {[id: string]: OptLabel}
}

export type OptLabel = string[]
