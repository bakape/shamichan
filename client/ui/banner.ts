export default () => {
	for (let id of ["options", "identity", "account"]) {
		highlightBanner(id)
	}
}

// Highlight options button by fading out and in, if no options are set
function highlightBanner(name: string) {
	const key = name + "_seen"
	if (localStorage.getItem(key)) {
		return
	}

	const el = document.querySelector('#banner-' + name)
	el.classList.add("blinking")

	el.addEventListener("click", () => {
		el.classList.remove("blinking")
		localStorage.setItem(key, '1')
	})
}
