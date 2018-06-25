// Forward a request to a client on a specific thread. If none found, execute
// onFail().
export async function forward(thread: number, msg: {}, onFail: () => void) {
	const cls: any[] = await (self as any).clients.matchAll({ type: "window" });
	let matched: any;
	for (let cl of cls) {
		if (new RegExp("/[\\w\\d]+/" + thread).test(cl.url)) {
			matched = cl;
			break;
		}
	}
	if (matched) {
		matched.postMessage(msg);
	} else {
		onFail();
	}
}
