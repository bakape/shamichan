// Helper functions for communicating with the server's JSON API

// Fetches and decodes a JSON response from the API. Returns a tuple of the
// fetched resource and error, if any
export async function fetchJSON<T>(url: string): Promise<[T, string]> {
	const res = await fetch(url)
	if (res.status !== 200) {
		return [null, await res.text()]
	}
	return [await res.json(), ""]
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

// Fetches HTML from the server. Returns a tuple of the fetched resource and
// error, if any
export async function fetchHTML(url: string): Promise<[string, string]> {
	const res = await fetch(url)
	if (res.status !== 200) {
		return ["", await res.text()]
	}
	return [await res.text(), ""]
}

// Fetch HTML of a board page
export async function fetchBoard(board: string): Promise<Response> {
	return await fetch(`/${board}/?minimal=true`)
}

// Fetch HTML of a thread page
export async function fetchThread(
	board: string,
	thread: number,
	lastN: number,
): Promise<Response> {
	let url = `/${board}/${thread}?minimal=true`
	if (lastN) {
		url += `&last=${lastN}`
	}
	return await fetch(url)
}
