import { message, send } from "./connection";
import { mine } from "./state";
import { getClosestID, on } from "./util";


export function sendPM(to: number, from: number) {
	const msg = prompt("Send a private message:");
	if (msg) {
		send(message.sendPM, {
			to, from,
			text: msg,
		})
	}
}


export function init() {
	on(
		document,
		"click",
		e => {
			const id = getClosestID(e.target as Element);
			const src = mine.values().next();
			if (id && src) {
				sendPM(id, src.value);
			}
		},
		{
			passive: true,
			selector: ".send-pm, .send-pm svg, .send-pm path",
		},
	);
}
