// Pointless service worker, so we can get Android Chrome native app install
// prompts.

import { open } from "../common/db"

self.addEventListener('install', () => { });
self.addEventListener('activate', async () => {
	await open();
});
