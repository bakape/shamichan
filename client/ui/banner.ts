import { BannerModal } from "../base"
import { write } from '../util'

export default () => {
	for (let id of ["options", "FAQ", "identity", "account"]) {
		highlightBanner(id)
	}
	new BannerModal(document.getElementById("FAQ"))
}

// Highlight options button by fading out and in, if no options are set
function highlightBanner(name: string) {
	const key = name + "_seen"
	if (localStorage.getItem(key)) {
		return
	}

	const el = document.querySelector('#banner-' + name)
	write(() =>
		el.classList.add("blinking"))

	el.addEventListener("click", () => {
		el.classList.remove("blinking")
		localStorage.setItem(key, '1')
	})
}
