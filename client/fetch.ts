// Helper functions for communicating with the server's JSON API

import { ThreadData } from "./posts/models"

// Single entry of the array, fetched through `/json/boardList`
export type BoardEntry = {
	id: string
	title: string
}

// Fetches and decodes a JSON response from the API. Returns a tuple of the
// fetched resource and error, if any
export async function fetchJSON<T>(url: string): Promise<[T, string]> {
	const res = await fetch(url)
	if (res.status !== 200) {
		return [null, await res.text()]
	}
	return [await res.json(), ""]
}

// Fetches HTML from the server. Returns a tuple of the fetched resource and
// error, if any
export async function fetchHTML(url: string): Promise<[string, string]> {
	const res = await fetch(url)
	if (res.status !== 200) {
		return ["", await res.text()]
	}
	return [await res.text(), ""]
}

// Send a POST request with a JSON body to the server
export async function postJSON(url: string, body: any): Promise<Response> {
	return await fetch(url, {
		method: "POST",
		body: JSON.stringify(body),
	})
}

// Send a POST request with a text body to the server
export async function postText(
	url: string,
	text: string,
): Promise<[string, string]> {
	const res = await fetch(url, {
		method: "POST",
		body: text,
	})
	const rt = await res.text()
	if (res.status === 200) {
		return [rt, ""]
	}
	return ["", rt]
}

// Fetch HTML of a board page
export async function fetchBoard(board: string): Promise<[string, string]> {
	return await fetchHTML(`/${board}/?noIndex=true`)
}

// Fetch thread JSON data
export async function fetchThread(
	board: string, thread: number, lastN: number,
): Promise<[ThreadData, string]> {
	let url = `/json/${board}/${thread}`
	if (lastN) {
		url += `?last=${lastN}`
	}
	return await fetchJSON<ThreadData>(url)
}
