// Client entry point

 // TODO: Remove, when proper structure done
import * as client from './client'
const c = client  // Prevents the compiler from removing as an unused import

import {displayLoading} from './state'
import {start as connect} from './connection'
import {loadFromDB, loadBoardConfig, page} from './state'
import {open} from './db'
import {renderBoard} from './page/board'
import BoardNavigation from './page/boardNavigation'
import {exec, defer} from './defer'
import bindThreadCreation from './posts/threadCreation'
import {fetchBoard} from './util'
import {write, $threads} from './render'

// Clear cookies, if versions mismatch.
const cookieVersion = 4
if (localStorage.getItem("cookieVersion") !== cookieVersion.toString()) {
	for (let cookie of document.cookie.split(";")) {
		const eqPos = cookie.indexOf("="),
			name = eqPos > -1 ? cookie.substr(0, eqPos) : cookie
		document.cookie = name + "=;expires=Thu, 01 Jan 1970 00:00:00 GMT"
	}
	localStorage.setItem("cookieVersion", cookieVersion.toString())
}

// Load all stateful modules in dependancy order
async function start() {
	// Load asynchronously and concurently as fast as possible
	const boardConf = loadBoardConfig(),
		boardData = fetchBoard(page.board)
	await open()
	await loadFromDB()
	await boardConf
	bindThreadCreation()
	new BoardNavigation()
	const html = renderBoard((await boardData).threads)
	write(() =>
		$threads.innerHTML = html)
	connect()
	exec()
	displayLoading(false)
}

start()
