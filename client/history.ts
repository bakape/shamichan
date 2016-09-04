// Inter-page navigation with HTML5 history

import {on, isMatch} from './util'
import {read, page, displayLoading} from './state'
import loadPage from './page/load'
import {synchronise} from './connection'
import {postSM, postEvent} from "./posts/posting/main"
import {scrollToAnchor} from "./scroll"
import {read as renderRead} from "./render"

// Bind event listener
on(document, "click", handleClick, {
	selector: "a.history, a.history img",
})

function handleClick(event: KeyboardEvent) {
	if (event.ctrlKey) {
		return
	}
	const href =
		((event.target as Element)
			.closest("a.history") as HTMLAnchorElement)
		.href
	navigate(href, event, true).catch(alertError)
}

// Navigate to the target og the URL and load its data. NewPoint indicates, if
// a new history state should be pushed.
async function navigate(url: string, event: Event, needPush: boolean) {
	let nextState = read(url)

	// Does the link point to the same page as this one?
	if (isMatch(nextState, page)) {
		return
	}
	if (event) {
		event.preventDefault()
	}

	displayLoading(true)

	// Load asynchronously and concurently as fast as possible
	let renderPage: () => void
	const ready = new Promise<void>((resolve) =>
		renderPage = resolve)
	const pageLoader = loadPage(nextState, ready)

	page.replaceWith(nextState)
	renderPage()
	await pageLoader
	postSM.feed(postEvent.reset)
	synchronise()

	if (needPush) {
		history.pushState(null, null, nextState.href)
	}
	displayLoading(false)
}

export default navigate

function alertError(err: Error) {
	displayLoading(false)
	alert(err)
	throw err
}

// For back and forward history events
window.onpopstate = (event: any) =>
	(navigate(event.target.location.href, null, false)
		.catch(alertError),
	renderRead(() =>
		scrollToAnchor()))
