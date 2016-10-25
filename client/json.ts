// Helper functions for communicating with the server's JSON API

import { BoardConfigs } from './state'
import { ThreadData } from "./posts/models"

// Data of a single board retrieved from the server through `/json/:board`
export type BoardData = {
	ctr: number
	threads: ThreadData[]
}

// Single entry of the array, fetched through `/json/boardList`
export type BoardEntry = {
	id: string
	title: string
}

// Fetches and decodes a JSON response from the API
export async function fetchJSON<T>(url: string): Promise<T> {
	const res = await fetch(url)
	await handleError(res)
	return await res.json()
}

// Send a POST request with a JSON body to the server
export async function postJSON(url: string, body: any): Promise<Response> {
	const res = await fetch(url, {
		method: "POST",
		body: JSON.stringify(body),
	})
	await handleError(res)
	return res
}

// Send a POST request with a text body to the server
export async function postText(url: string, text: string): Promise<string> {
	const res = await fetch(url, {
		method: "POST",
		body: text,
	})
	await handleError(res)
	return await res.text()
}

// Throw the status text of a Response as an error on HTTP errrors
export async function handleError(res: Response) {
	if (!res.ok) {
		throw new Error(await res.text())
	}
}

// Returns a list of all boards created in alphabetical order
export async function fetchBoardList(): Promise<BoardEntry[]> {
	return (await fetchJSON<BoardEntry[]>("/json/boardList"))
		.sort((a, b) =>
			a.id.localeCompare(b.id))
}

// Fetch configurations of a specific board
export async function fetchBoarConfigs(board: string): Promise<BoardConfigs> {
	return await fetchJSON<BoardConfigs>(`/json/boardConfig/${board}`)
}

// Fetch JSON data of a board page
export async function fetchBoard(board: string): Promise<BoardData> {
	return await fetchJSON<BoardData>(`/json/${board}/`)
}

// Fetch thread JSON data
export async function fetchThread(
	board: string, thread: number, lastN: number,
): Promise<ThreadData> {
	let url = `/json/${board}/${thread}`
	if (lastN) {
		url += `?last=${lastN}`
	}
	return await fetchJSON<ThreadData>(url)
}
