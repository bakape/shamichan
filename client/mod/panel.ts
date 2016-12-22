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

		this.el.querySelector("form").addEventListener("submit", e =>
			this.onSubmit(e))

		const toggle = (this.el
			.querySelector(`input[name="showCheckboxes"]`) as HTMLInputElement)
		toggle.addEventListener("change", e =>
			toggleCheckboxDisplay((event.target as HTMLInputElement).checked))

		toggleCheckboxDisplay(displayCheckboxes)
		write(() => {
			this.el.style.display = "inline-block"
			toggle.checked = displayCheckboxes
		})
	}

	private onSubmit(e: Event) {
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

		const action = (this.el
			.querySelector(`select[name="action"]`) as HTMLInputElement)
			.value
		switch (action) {
			case "deletePost":
				this.deletePost(models)
				break
		}
	}

	// Deleted one or multiple selected posts
	private deletePost(models: Post[]) {
		this.postJSON("/admin/deletePost", {
			ids: models.map(m =>
				m.id),
			board: page.board,
		})
	}

	// Post JSON to server and handle errors
	private async postJSON(url: string, data: {}) {
		extend(data, newRequest())
		const res = await postJSON(url, data)
		if (res.status !== 200) {
			throw await res.text()
		}
	}
}

function toggleCheckboxDisplay(on: boolean) {
	localStorage.setItem("hideModCheckboxes", (!on).toString())
	checkboxStyler(on)
}
