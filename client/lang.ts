// Provides type-safe and selective mappings for the language packs

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
