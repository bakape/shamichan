import { View } from "../base"
import { postJSON, toggleHeadStyle, getClosestID } from "../util"
import collectionView from "../posts/collectionView"
import { ModerationLevel } from "../common"

type ModerationData = {
	id: number;
	ban?: BanData;
	censor?: CensorData;
}

type CensorData = {
	byIP: boolean;
	delPost?: boolean;
	spoil?: boolean;
	delImg?: boolean;
	purge?: {
		isSet: boolean;
		reason: string;
	};
}

type BanData = {
	isSet: boolean;
	global: boolean;
	shadow: boolean;
	duration: number;
	reason: string;
}

let displayCheckboxes = localStorage.getItem("hideModCheckboxes") !== "true",
	checkboxStyler: (toggle: boolean) => void

// Moderation panel with various post moderation and other controls
export default class ModPanel extends View<null> {
	constructor(position: ModerationLevel) {
		checkboxStyler = toggleHeadStyle(
			"mod-checkboxes",
			".mod-checkbox{ display: inline; }"
		);

		super({ el: document.getElementById("moderation-panel") });

		document.getElementById("meidovision").addEventListener("click", () => {
			this.viewAllByIP();
		});

		if (position == ModerationLevel.admin) {
			document.getElementById("redirect-ip").addEventListener("click", () => {
				this.redirectIP();
			});

			document.getElementById("notification").addEventListener("click", () => {
				this.sendNotification();
			});
		}

		this.el.querySelector("form").addEventListener("submit", e => {
			this.onSubmit(e);
		});

		this.inputElement("clear").addEventListener("click", () => {
			this.clear();
		});

		const checkboxToggle = this.inputElement("showCheckboxes");
		checkboxToggle.checked = displayCheckboxes;
		checkboxToggle.addEventListener("change", e => {
			this.setVisibility((e.target as HTMLInputElement).checked);
		}, { passive: true });

		this.setVisibility(displayCheckboxes);
	}

	// Returns a reference to the post marked for moderation
	private getChecked(): HTMLInputElement {
		return document.querySelector(".mod-checkbox:checked") as HTMLInputElement;
	}

	private setVisibility(on: boolean) {
		localStorage.setItem("hideModCheckboxes", (!on).toString());
		this.setSlideOut(on);
		checkboxStyler(on);
	}

	// Create display of all posts made by selected post's author
	private async viewAllByIP() {
		const checked = this.getChecked();
		if (!checked) {
			return;
		}
		const id = getClosestID(checked);

		const res = await postJSON(`/api/same-IP/${id}`, null);
		if (res.status !== 200) {
			this.el.querySelector(".form-response").textContent =
				await res.text();
			return;
		}

		const posts = await res.json();
		if (posts) {
			new collectionView(posts);
		}
	}

	// Redirect a poster to a specified URL
	private async redirectIP() {
		const checked = this.getChecked();
		if (!checked) {
			return;
		}
		const id = getClosestID(checked);

		const url = (document
			.getElementById("redirect-location") as HTMLInputElement).value;
		await this.postJSON("/api/redirect/by-ip", { id, url });
	}

	// Send a notification to all connected clients
	private async sendNotification() {
		const text = (document
			.getElementById("notification-text") as HTMLInputElement).value;
		await this.postJSON("/api/notification", text);
	}

	// Parse and send moderation data to server
	private async onSubmit(e: Event) {
		e.preventDefault();
		e.stopImmediatePropagation();

		const errlog = this.el.querySelector(".form-response");
		errlog.textContent = "";

		const checked = this.getChecked();
		if (!checked) {
			return;
		}
		const id = getClosestID(checked);
		let data: ModerationData = {
			id: id,
		}

		const banform = this.inputElement("ban-poster");
		if (banform && banform.checked) {
			const ban = this.parseBan();
			if (ban.err) {
				errlog.textContent = ban.err;
				return;
			}
			if (ban.data) {
				data.ban = ban.data;
			}
		}

		const censor = this.parseCensor();
		if (censor.err) {
			errlog.textContent = censor.err;
			return;
		}
		if (censor.data) {
			data.censor = censor.data;
		}

		if (data.ban || data.censor) {
			await this.postJSON("/api/moderate", data);
		}

		checked.checked = false;
	}

	private parseBan(): { data: BanData, err: string } {
		const dur = this.extractDuration();
		if (!dur) {
			return { data: null, err: "No ban duration" };
		}

		const r = this.inputElement("ban-reason").value;
		if (r === "") {
			return { data: null, err: "No ban reason" };
		}

		const data: BanData = {
			isSet: true,
			global: false,
			shadow: this.inputElement("shadow").checked,
			duration: dur,
			reason: r,
		}

		// Global checkbox doesn't always exist
		const g = this.inputElement("global");
		if (g) {
			data.global = g.checked;
		}

		return { data: data, err: null }
	}

	private parseCensor(): { data: CensorData, err: string } {
		let data: CensorData = {
			byIP: this.inputElement("all").checked,
		}
		let set = false;

		// Only send most powerful request.
		// For example, no need to spoiler an image if it will be purged
		// Purge deletes the post and deletes attached file, if any
		const purge = this.inputElement("purge-post");
		if (purge && purge.checked) {
			const r = this.inputElement("purge-reason").value;
			if (!r) {
				return { data: null, err: "Missing purge reason" };
			}
			data.purge = { isSet: true, reason: r }
			set = true;
		}
		else {
			if (this.inputElement("delete-post").checked) {
				data.delPost = true;
				set = true;
			}

			if (this.inputElement("delete-image").checked) {
				data.delImg = true;
				set = true;
			}
			else if (this.inputElement("spoiler-image").checked) {
				data.spoil = true;
				set = true;
			}
		}
		if (!set) {
			data = null;
		}

		return { data: data, err: null }
	}

	// Restore panel to its default state
	private clear() {
		const checked = this.getChecked();
		if (checked) {
			checked.checked = false;
		}
		for (const e of this.el.querySelectorAll("input")) {
			switch (e.type) {
				case "number":
				case "text":
					e.value = "";
					break;
				case "checkbox":
					if (e.name === "showCheckboxes") {
						continue;
					}
					e.checked = false;
					break;
			}
		}
		this.el.querySelector(".form-response").textContent = "";
	}

	// Post JSON to server and handle errors
	private async postJSON(url: string, data: {}) {
		const res = await postJSON(url, data);
		this.el.querySelector(".form-response").textContent =
			res.status === 200
				? ""
				: await res.text();
	}

	// Force panel to stay visible
	public setSlideOut(on: boolean) {
		this.el.classList.toggle("keep-visible", on);
	}
}
