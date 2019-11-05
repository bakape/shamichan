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
		credentials: 'include',
		body: JSON.stringify(body),
	})
}

// Fetch HTML of a board page
export async function fetchBoard(
	board: string,
	page: number,
	catalog: boolean,
): Promise<Response> {
	return fetch(
		`/${board}/${catalog ? "catalog" : ""}?minimal=true&page=${page}`)
}
