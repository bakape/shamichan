import { View } from "../base"
import { postJSON, toggleHeadStyle, trigger } from "../util"
import { Post } from "../posts"
import { getModel, config } from "../state"

let displayCheckboxes = localStorage.getItem("hideModCheckboxes") !== "true",
	checkboxStyler: (toggle: boolean) => void

// Moderation panel with various post moderation and other controls
export default class ModPanel extends View<null> {
	constructor() {
		checkboxStyler = toggleHeadStyle(
			"mod-checkboxes",
			".mod-checkbox{ display: inline; }"
		)

		super({ el: document.getElementById("moderation-panel") })
		new BanForm()
		new NotificationForm()
		new PostPurgeForm();
		new SpamHandlerForm();

		this.el.querySelector("form").addEventListener("submit", e =>
			this.onSubmit(e))

		this.el
			.querySelector("select[name=action]")
			.addEventListener("change", () => this.onSelectChange(), {
				passive: true
			})
		this.inputElement("clear")
			.addEventListener("click", () => {
				for (let el of this.getChecked()) {
					el.checked = false
				}
			},
				{ passive: true })

		const checkboxToggle = this.inputElement("showCheckboxes")
		checkboxToggle.checked = displayCheckboxes
		checkboxToggle.addEventListener(
			"change",
			e =>
				this.setVisibility((event.target as HTMLInputElement).checked),
			{ passive: true },
		)

		this.setVisibility(displayCheckboxes)
	}

	private setVisibility(on: boolean) {
		localStorage.setItem("hideModCheckboxes", (!on).toString())
		this.setSlideOut(on)
		checkboxStyler(on)
	}

	private async onSubmit(e: Event) {
		e.preventDefault()
		e.stopImmediatePropagation()

		const checked = this.getChecked(),
			models = [...checked].map(getModel)

		// Send multiple requests with post ID to server
		const sendIDRequests = async (formID: string, url: string) => {
			if (!checked.length) {
				return;
			}
			const args = HidableForm.forms[formID].vals();
			for (let id of mapToIDs(models)) {
				args["id"] = id;
				await this.postJSON(url, args);
			}
		}

		// Send request with post IDs to server
		const sendMultiIDRequest = async (
			path: string,
			withImages: boolean,
		) => {
			if (checked.length) {
				await this.postJSON(
					"/api" + path,
					mapToIDs(
						withImages
							? models.filter(m => !!m.image)
							: models
					),
				);
			}
		};

		switch (this.getMode()) {
			case "deletePost":
				await sendMultiIDRequest("/delete-posts", false);
				break;
			case "spoilerImage":
				await sendMultiIDRequest("/spoiler-image", true);
				break;
			case "deleteImage":
				await sendMultiIDRequest("/delete-image", true);
				break;
			case "ban":
				if (checked.length) {
					const args = HidableForm.forms["ban"].vals();
					args["ids"] = mapToIDs(models);
					this.postJSON("/api/ban", args);
				}
				break;
			case "purgePost":
				await sendIDRequests("purgePost", "/api/purge-post");
				break;
			case "handleSpam":
				for (let p of this.getChecked().map(getModel)) {
					await postJSON("/api/delete-posts/by-ip", {
						id: p.id,
						duration: 60 * 24 * 30, // 30 days
						reason: "spam",
					});
				}
				break;
			case "notification":
				const f = HidableForm.forms["notification"]
				await this.postJSON("/api/notification", f.vals())
				f.clear()
				break
		}

		for (let el of checked) {
			el.checked = false
		}
	}

	// Get selected post checkboxes
	private getChecked(): HTMLInputElement[] {
		const query = document.querySelectorAll(".mod-checkbox:checked")
		var els = new Array(query.length)

		for (let i = 0; i < query.length; i++) {
			els[i] = query[i]
		}

		return els
	}

	// Return current action mode
	private getMode(): string {
		return (this.el
			.querySelector(`select[name="action"]`) as HTMLInputElement)
			.value
	}

	// Post JSON to server and handle errors
	private async postJSON(url: string, data: {}) {
		const res = await postJSON(url, data)
		this.el.querySelector(".form-response").textContent =
			res.status === 200
				? ""
				: await res.text()
	}

	// Change additional input visibility on action change
	private onSelectChange() {
		HidableForm.show(this.getMode())
	}

	// Force panel to stay visible
	public setSlideOut(on: boolean) {
		this.el.classList.toggle("keep-visible", on)
	}
}

abstract class HidableForm extends View<null> {
	public static forms: { [id: string]: HidableForm } = {}
	public abstract vals(): any

	constructor(id: string) {
		super({ el: document.getElementById(id + "-form") })
		HidableForm.forms[id] = this
		this.toggleDisplay(false)
	}

	public toggleDisplay(on: boolean) {
		for (let el of this.el.getElementsByTagName("input")) {
			el.disabled = !on
		}
		this.el.classList.toggle("hidden", !on)
	}

	// Hide all displayed forms
	public static hideAll() {
		for (let id in HidableForm.forms) {
			HidableForm.forms[id].toggleDisplay(false)
		}
	}

	// Show a form by ID, if any
	public static show(id: string) {
		HidableForm.hideAll()
		const f = HidableForm.forms[id]
		if (f) {
			f.toggleDisplay(true)
		}
	}

	// Clear any text inputs
	public clear() {
		for (let el of this.el.querySelectorAll("input[type=text]")) {
			(el as HTMLInputElement).value = ""
		}
	}
}

// Ban input fields
class BanForm extends HidableForm {
	constructor() {
		super("ban")
	}

	// Get input field values
	public vals(): { [key: string]: any } {
		const data = {
			duration: this.extractDuration(),
			reason: this.inputElement("reason").value,
		}
		const g = this.inputElement("global")
		if (g) {
			data["global"] = g.checked
		}
		return data
	}
}

class PostPurgeForm extends HidableForm {
	constructor() {
		super("purgePost");
	}

	// Get input field values
	public vals(): { [key: string]: any } {
		return {
			reason: this.inputElement("purge-reason").value,
		};
	}
}

// Form for sending notifications to all connected clients
class NotificationForm extends HidableForm {
	constructor() {
		super("notification")
	}

	public vals(): string {
		return this.inputElement("notification").value
	}
}

// Shorthand to handle spam
class SpamHandlerForm extends HidableForm {
	constructor() {
		super("handlesSpam")
	}

	public vals(): string {
		return null;
	}
}

function mapToIDs(models: Post[]): number[] {
	return models.map(m =>
		m.id)
}
