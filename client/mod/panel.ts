// Moderation panel with various post moderation and other controls

import View from "../view"
import { write, threads } from "../render"
import { Post } from "../posts/models"
import { getModel, page } from "../state"
import { newRequest } from "./common"
import { extend } from "../util"
import { postJSON } from "../fetch"
import { toggleHeadStyle } from "../options/specs"

let panel: ModPanel,
	banInputs: BanInputs,
	displayCheckboxes = localStorage.getItem("hideModCheckboxes") !== "true"

const checkboxStyler = toggleHeadStyle(
	"mod-checkboxes",
	".mod-checkbox{ display: inline; }"
)

export default class ModPanel extends View<null> {
	constructor() {
		if (panel) {
			return
		}
		super({ el: document.getElementById("moderation-panel") })
		panel = this
		banInputs = new BanInputs()

		this.el.querySelector("form").addEventListener("submit", e =>
			this.onSubmit(e))

		this.el
			.querySelector("select[name=action]")
			.addEventListener("change", () => this.onSelectChange(), {
				passive: true
			})

		const toggle = (this.el
			.querySelector(`input[name="showCheckboxes"]`) as HTMLInputElement)
		toggle.addEventListener("change", e =>
			toggleCheckboxDisplay((event.target as HTMLInputElement).checked))

		toggleCheckboxDisplay(displayCheckboxes)
		write(() => {
			this.el.style.display = "inline-block"
			toggle.checked = displayCheckboxes
			document
				.querySelector("#identity > table tr:first-child")
				.style
				.display = "table-row"
		})
	}

	private async onSubmit(e: Event) {
		e.preventDefault()
		e.stopImmediatePropagation()

		const checked = (threads
			.querySelectorAll(".mod-checkbox:checked") as HTMLInputElement[])
		if (!checked.length) {
			return
		}
		const models = new Array<Post>(checked.length)
		for (let i = 0; i < checked.length; i++) {
			const el = checked[i]
			models[i] = getModel(el)
			el.checked = false
		}

		switch (this.getMode()) {
			case "deletePost":
				await this.deletePost(models)
				break
			case "ban":
				await this.ban(models)
				break
		}

		for (let el of checked) {
			el.checked = false
		}
	}

	// Return current action mode
	private getMode(): string {
		return (this.el
			.querySelector(`select[name="action"]`) as HTMLInputElement)
			.value
	}

	// Deleted one or multiple selected posts
	private async deletePost(models: Post[]) {
		await this.postJSON("/admin/deletePost", {
			ids: models.map(m =>
				m.id),
			board: page.board,
		})
	}

	// Ban selected posts
	private async ban(models: Post[]) {
		const args = {
			ids: models.map(m =>
				m.id),
			board: page.board,
		}
		extend(args, banInputs.vals())

		await this.postJSON("/admin/ban", args)
		banInputs.clear()
	}

	// Post JSON to server and handle errors
	private async postJSON(url: string, data: {}) {
		extend(data, newRequest())
		const res = await postJSON(url, data)
		if (res.status !== 200) {
			throw await res.text()
		}
	}

	// Change additional input visibility on action change
	private onSelectChange() {
		banInputs.toggleDisplay(this.getMode() === "ban")
	}
}

function toggleCheckboxDisplay(on: boolean) {
	localStorage.setItem("hideModCheckboxes", (!on).toString())
	checkboxStyler(on)
}

// Ban input fields
class BanInputs extends View<null> {
	constructor() {
		super({ el: document.getElementById("ban-form") })
	}

	public toggleDisplay(on: boolean) {
		write(() => {
			(this.el
				.querySelector("input[name=reason]") as HTMLInputElement)
				.disabled = !on
			this.el.classList.toggle("hidden", !on)
		})
	}

	// Clear values of all fields
	public clear() {
		write(() => {
			for (let el of this.el.getElementsByTagName("input")) {
				el.value = ""
			}
		})
	}

	// Get input field values
	public vals(): { [key: string]: any } {
		let duration = 0
		for (let el of this.el.querySelectorAll("input[type=number]")) {
			let times = 1
			switch (el.getAttribute("name")) {
				case "day":
					times *= 24
				case "hour":
					times *= 60
			}
			const val = parseInt((el as HTMLInputElement).value)
			if (val) { // Empty string parses to NaN
				duration += val * times
			}
		}

		return {
			duration,
			reason: (this.el
				.querySelector("input[name=reason]") as HTMLInputElement)
				.value
		}
	}
}
