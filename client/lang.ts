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
export const {
	posts, banner, images, navigation, reports, time, sync, syncwatch, opts
} = lang


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
	modes: {
		small: string
		sharp: string
		hide: string
		none: string
		full: string
		width: string
		height: string
		both: string
	}
	importConfig: {
		done: string
		corrupt: string
	}
	langApplied: string
	labels: {
		export: OptLabel
		import: OptLabel
		hidden: OptLabel
		lang: OptLabel
		inlineFit: OptLabel
		thumbs: OptLabel
		imageHover: OptLabel
		webmHover: OptLabel
		autogif: OptLabel
		spoilers: OptLabel
		notifications: OptLabel
		anonymise: OptLabel
		relativeTime: OptLabel
		nowPlaying: OptLabel
		illyaBGToggle: OptLabel
		illyaMuteToggle: OptLabel
		horizontalPosting: OptLabel
		theme: OptLabel
		userBG: OptLabel
		userBGimage: OptLabel
		lastn: OptLabel
		postUnloading: OptLabel
		alwaysLock: OptLabel
		new: OptLabel
		toggleSpoiler: OptLabel
		textSpoiler: OptLabel
		done: OptLabel
		expandAll: OptLabel
		workMode: OptLabel
		google: OptLabel
		iqdb: OptLabel
		saucenao: OptLabel
		desustorage: OptLabel
		exhentai: OptLabel
	}
}

type OptLabel = string[]
