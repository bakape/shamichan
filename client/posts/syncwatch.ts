import lang from "../lang"
import { pad } from "../util"
import { handlers, message } from "../connection"

const threads = document.getElementById("threads")

let offset = 0

handlers[message.serverTime] = (time: number) =>
	offset = Math.floor(Date.now() / 1000) - time

// Synchronized time counter for things like watching animu together and such
class Syncwatch {
	private el: HTMLElement
	private hour: number
	private min: number
	private sec: number
	private start: number
	private end: number

	constructor(el: HTMLElement) {
		this.el = el
		this.el.classList.add("ticking")
		for (let id of ["hour", "min", "sec", "start", "end"]) {
			this[id] = parseInt(this.el.getAttribute("data-" + id))
		}
		this.render()
	}

	private render() {
		const now = Math.round(Date.now() / 1000) + offset
		if (now > this.end) {
			this.el.innerText = lang.syncwatch["finished"]
			return
		} else if (now < this.start) {
			this.el.innerHTML = (this.start - now).toString()
		} else {
			let diff = now - this.start
			const hour = Math.floor(diff / 3600)
			diff -= hour * 3600
			const min = Math.floor(diff / 60)
			diff -= min * 60
			this.el.innerHTML = this.formatTime(hour, min, diff)
				+ " / "
				+ this.formatTime(this.hour, this.min, this.sec)
		}

		setTimeout(() => {
			if (threads.contains(this.el)) {
				this.render()
			}
		}, 1000)
	}

	private formatTime(hour: number, min: number, sec: number): string {
		return `${pad(hour)}:${pad(min)}:${pad(sec)}`
	}
}

// Find and start any non-running synchronized time counters
export function findSyncwatches(ns: NodeSelector) {
	for (let el of ns.querySelectorAll(".syncwatch:not(.ticking)")) {
		new Syncwatch(el)
	}
}
