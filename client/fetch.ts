// Utility functions for retrieving resources from the server

import {BoardConfigs} from './state'
import {ThreadData} from './posts/models'

// Single entry of the array, fetched through `/json/boardList`
export type BoardEntry = {
	id: string
	title: string
}

// Data of a single board retrieved from the server through `/json/:board`
export type BoardData = {
	ctr: number
	threads: ThreadData[]
}

// Fetches and decodes a JSON response from the API
export const fetchJSON = async (url: string): Promise<any> =>
	await (await fetch(url)).json()

// Returns a list of all boards created in alphabetical order
export const fetchBoardList = async (): Promise<BoardEntry[]> =>
	((await fetchJSON("/json/boardList") as BoardEntry[]))
	.sort((a, b) =>
		a.id.localeCompare(b.id))

// Fetch configurations of a specific board
export const fetchBoarConfigs = async (board: string): Promise<BoardConfigs> =>
	await fetchJSON(`/json/boardConfig/${board}`)

// Fetch board contents from the server
export const fetchBoard = async (board: string): Promise<BoardData> =>
	await fetchJSON(`/json/${board}`)
