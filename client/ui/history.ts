// Inter-page navigation with HTML5 history

import { on, scrollToAnchor } from '../util'
import { read, page, displayLoading } from '../state'
import { loadPage } from '../page'
import { postSM, postEvent } from "../posts"
import { synchronise, connSM, connState } from "../connection"
import options from "../options"

// Handle a click on any .history anchor
function handleClick(event: KeyboardEvent) {
	// Don't trigger, when user is trying to open in a new tab
	const bypass = event.which !== 1
		|| event.ctrlKey
		|| connSM.state !== connState.synced
	if (bypass) {
		return
	}

	let target = event.target as Element

	if (target.classList.contains("post-link") && options.postInlineExpand) {
		return
	}

	if (target.classList.contains("hash-link")) {
		target = target.closest("em").firstElementChild
		location.hash = "#p" + target.getAttribute("data-id")
	}

	const href = (target.closest("a.history") as HTMLAnchorElement).href
	navigate(href, event, true).catch(alertError)
}

// Navigate to the target og the URL and load its data. NewPoint indicates, if
// a new history state should be pushed.
export default async function navigate(
	url: string,
	event: Event,
	needPush: boolean,
) {
	let nextState = read(url)

	// Does the link point to the same page as this one?
	let isSame = true
	for (let key of ["thread", "lastN", "board", "catalog"]) {
		if (nextState[key] !== page[key]) {
			isSame = false
			break
		}
	}
	if (isSame) {
		// Soft reload the page
		if (event && (event.target as Element).classList.contains("reload")) {
			needPush = false
		} else {
			return scrollToAnchor()
		}
	}

	if (event) {
		event.preventDefault()
	}
	if (connSM.state !== connState.synced) {
		return
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
		scrollToAnchor()
		history.pushState(null, null, nextState.href)
	}

	displayLoading(false)
}

function alertError(err: Error) {
	displayLoading(false)
	alert(err)
	throw err
}

// Bind event listener
on(document, "click", handleClick, {
	selector: "a.history, a.history img, .hash-link",
})

// For back and forward history events
window.onpopstate = e =>
	navigate((e.target as Window).location.href, null, false)
		.catch(alertError)

