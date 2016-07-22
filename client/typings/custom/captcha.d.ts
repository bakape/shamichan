// Solve Media AJAX API controler
// https://portal.solvemedia.com/portal/help/pub/ajax
declare class ACPuzzleController  {
	create(key: string, elID: string, opts?: ACPuzzleOptions): ACPuzzleController
	destroy(): void
	get_challenge(): string
	get_response(): string
	reload(): void
}

interface ACPuzzleOptions {
	multi: boolean
	id: string
	theme: string
}

interface Window {
	ACPuzzle: ACPuzzleController
}

declare var ACPuzzle: typeof window.ACPuzzle
