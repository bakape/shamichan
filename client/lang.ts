// Provides type-safe and selective mappings for the language packs

import {makeEl, HTML, fetchJSON} from './util'
import {write} from './render'
import {defer} from './defer'
import options from './options'

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
}

const lang = (window as any).lang as LanguagePack

// Export each container indivudually for namespacing purposes
// Can't use destructuring, because it breaks with the SystemJS module compiler.
export const posts = lang.posts
export const ui = lang.ui
export const banner = lang.banner
export const images = lang.images
export const navigation = lang.navigation
export const reports = lang.reports
export const time = lang.time
export const sync = lang.sync
export const syncwatch = lang.syncwatch
export const mod = lang.mod
export const opts = lang.opts
export let admin: LnAdmin

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
	[index: string]: string
}

type LnUI = {
	cancel: string
	done: string
	send: string
	add: string
	apply: string
	search: string
	invalidCaptcha: string
	focusForCaptcha: string
	reloadCaptcha: string
	[index: string]: string
}

type LnBanner = {
	showSeconds: string
	worksBestWith: string
	name: string
	email: string
	options: string
	identity: string
	acccount: string
	FAQ: string
	feedback: string
	googleSong: string
	sync: string
	[index: string]: string
}

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
	minute: string[]
	hour: string[]
	day: string[]
	month: string[]
	year: string[]
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
	submit: string
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

type LnAdmin = {
	boardNameTaken: string
	[index: string]: OptLabel|string
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

// Fetch the administrator language pack
export const fetchAdminPack = async (): Promise<LnAdmin> =>
	admin = admin || await fetchJSON(`/assets/lang/${options.lang}/admin.json`)
