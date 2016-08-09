// Client entry point

 // TODO: Remove, when proper structure done
import * as client from './client'
const c = client  // Prevents the compiler from removing as an unused import

import {displayLoading} from './state'
import {start as connect} from './connection'
import {loadFromDB, page, boardConfig} from './state'
import {open} from './db'
import renderBoard from './page/board'
import BoardNavigation from './page/boardNavigation'
import {exec, defer} from './defer'
import bindThreadCreation from './posts/posting/threadCreation'
import bindHistory from './history'
import {write, $threads} from './render'
import {fetchBoarConfigs, fetchBoard} from './fetch'

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
	const {board} = page,
		boardConf = fetchBoarConfigs(board),
		boardData = fetchBoard(board)
	await open()
	await loadFromDB()
	const conf = await boardConf,
		data = await boardData
	new BoardNavigation()
	boardConfig.replaceWith(conf)
	const html = renderBoard(data.threads)
	write(() =>
		$threads.innerHTML = html)
	bindThreadCreation()
	bindHistory()
	connect()
	exec()
	displayLoading(false)
}

start()
