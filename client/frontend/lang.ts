// Provides type-safe and selective mappings for the language packs.
// Must not use imports, to preserve load order.

import { LanguagePack } from "../common/types";

export default JSON.parse(
	document
		.getElementById("lang-data")
		.textContent
) as LanguagePack
