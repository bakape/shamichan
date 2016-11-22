// Provides type-safe and selective mappings for the language packs

import {makeEl, HTML} from './util'
import {write} from './render'
import {defer} from './defer'

type LanguagePack = {
	posts: LnPosts
	ui: LnUI
	banner: LnBanner
	images: LnImages
	navigation: LnNavigation
	reports: LnReports
	time: LnTime
	sync: string[]
	syncwatch: LnSyncwatch
	mod: LnMod
	opts: LnOpts
	identity: LnIdentity
}

const lang = (window as any).lang as LanguagePack

// Export each container individually for namespacing purposes
// Can't use destructuring, because it breaks with the SystemJS module compiler.
export const posts = lang.posts,
	ui = lang.ui,
	banner = lang.banner,
	images = lang.images,
	navigation = lang.navigation,
	reports = lang.reports,
	time = lang.time,
	sync = lang.sync,
	syncwatch = lang.syncwatch,
	mod = lang.mod,
	opts = lang.opts,
	identity = lang.identity

type StringTuple = [string, string]

type LnPosts = {
	anon: string
	newThread: string
	reply: string
	you: string
	OP: string
	locked: string
	uploading: string
	subject: string
	uploadProgress: string
	threadLocked: string
	quoted: string
	board: string
	spoiler: string
	and: string
	omitted: string
	unfinishedPost: string
	post: StringTuple
	image: StringTuple
	thumbnailing: string
	[index: string]: any
}

type LnUI = {
	cancel: string
	done: string
	send: string
	add: string
	mustMatch: string
	apply: string
	search: string
	invalidCaptcha: string
	focusForCaptcha: string
	reloadCaptcha: string
	submit: string
	rules: string
	close: string
	showNotice: string
	sortMode: string
	searchTooltip: string
	refresh: string
	sessionExpired: string
	[index: string]: any
}

type LnBanner = {
	worksBestWith: string
	options: string
	identity: string
	account: string
	FAQ: string
	feedback: string
	googleSong: string
	sync: string
	[index: string]: string
}

type LnIdentity = {[name: string]: StringTuple}

type LnImages = {
	show: string
	hide: string
	expand: string
	contract: string
	[index: string]: string
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
	[index: string]: string
}

type LnReports = {
	post: string
	reporting: string
	submitted: string
	setup: string
	leadError: string
	[index: string]: string
}

type LnTime = {
	week: string[]
	calendar: string[]
	justNow: string
	minute: StringTuple
	hour: StringTuple
	day: StringTuple
	month: StringTuple
	year: StringTuple
	in: string
	ago: string
	[index: string]: string|string[]
}

type LnSyncwatch = {
	starting: string
	finished: string
	[index: string]: string
}

type LnMod = {
	id: string
	register: string
	logout: string
	logoutAll: string
	password: string
	repeat: string
	changePassword: string
	oldPassword: string
	newPassword: string
	mustMatch: string
	nameTaken: string
	wrongCredentials: string
	wrongPassword: string
	theFuck: string
	configureServer: string
	createBoard: string
	configureBoard: string
	[index: string]: string
}

type LnOpts = {
	tabs: string[]
	modes: {[mode: string]: string}
	importConfig: {
		done: string
		corrupt: string
	}
	langApplied: string
	labels: {[id: string]: StringTuple}
}

// Load language-specific CSS
function languageCSS() {
	const el = makeEl(HTML
		`<style>
			.locked:after {
				content: "${posts.threadLocked}";
			}
			.locked > header nav:after {
				content: " (${posts.locked})";
			}
		</style>`)
	write(() =>
		document.head.appendChild(el))
}

defer(languageCSS)
