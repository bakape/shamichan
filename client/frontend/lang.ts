// Provides type-safe and selective mappings for the language packs.
// Must not use imports, to preserve load order.

type LanguagePack = {
	posts: { [key: string]: string }
	plurals: { [key: string]: [string, string] }
	time: {
		calendar: string[]
		week: string[]
	}
	ui: { [key: string]: string }
	sync: string[]
}

export default JSON.parse(
	document
		.getElementById("lang-data")
		.textContent
) as LanguagePack
