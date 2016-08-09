// Inter-page navigation with HTML5 history

import {on, isMatch} from './util'
import {read, page, displayLoading, boardConfig, setSyncCounter} from './state'
import renderBoard from './page/board'
import {fetchBoard, fetchBoarConfigs} from './fetch'
import {write, $threads} from './render'
import {synchronise} from './connection'

// Bind event listener
export default () =>
	on(document, "click", handleClick, {
		selector: "a.history",
	})

const handleClick = (event: KeyboardEvent) =>
	!event.ctrlKey
	&& navigate((event.target as any).href, event).catch(alertError)

// Navigate to the target og the URL and load its data. NewPoint indicates, if
// a new history state should be pushed.
async function navigate(url: string, event: Event) {
	let nextState = read(url)

	// Does the link point to the same page as this one?
	if (isMatch(nextState, page)) {
		return
	}
	if (event) {
		event.preventDefault()
	}

	displayLoading(true)

	// TODO: Diferentiate board and thread loading logic
	const {board} = nextState,
		fConf = fetchBoarConfigs(board),
		fData = fetchBoard(board),
		conf = await fConf,
		data = await fData,
		html = renderBoard(board, conf, data.threads)
	page.replaceWith(nextState)
	boardConfig.replaceWith(conf)

	setSyncCounter(0)
	synchronise()
	write(() =>
		$threads.innerHTML = html)

	if (event) {
		history.pushState(null, null, nextState.href)
	}
	displayLoading(false)
}

function alertError(err: Error) {
	displayLoading(false)
	alert(err)
}

// For back and forward history events
window.onpopstate = (event: any) =>
	navigate(event.target.location.href, null).catch(alertError)
