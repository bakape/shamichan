// Provides type-safe and selective mappings for the language packs

import { makeEl, HTML } from './util'
import { write } from './render'

type LanguagePack = {
	posts: { [key: string]: string }
	plurals: { [key: string]: [string, string] }
	time: {
		calendar: string[]
		week: string[]
	}
	ui: { [key: string]: string }
	sync: string[]
	syncwatch: { [key: string]: string }
	opts: {
		importConfig: { [key: string]: string }
		langApplied: string
	}
}

const lang = (window as any).lang as LanguagePack
export default lang

// Load language-specific CSS
{
	const el = makeEl(HTML
		`<style>
			.locked:after {
				content: "${lang.ui["threadLocked"]}";
			}
			.locked > header nav:after {
				content: " (${lang.posts["locked"]})";
			}
		</style>`)
	write(() =>
		document.head.appendChild(el))
}
