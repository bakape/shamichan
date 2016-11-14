import View from '../view'
import Model from '../model'
import { write } from '../render'
import { HTML, makeFrag } from '../util'
import { fetchHTML } from "../fetch"

const selected = new Set<string>(),
	panel = document.getElementById("left-panel"),
	spacer = document.getElementById("left-spacer")

let navigation: BoardNavigation,
	selectionPanel: BoardSelectionPanel,
	lastPanelWidth: number

// View for navigating between boards and selecting w
class BoardNavigation extends View<Model> {
	constructor() {
		super({ el: document.getElementById("board-navigation") })
		this.render()
		this.onClick({
			".board-selection": e =>
				this.togglePanel(e.target as Element),
		})
	}

	public render() {
		let html = "["
		const boards = ["all", ...Array.from(selected).sort()]
		for (let i = 0; i < boards.length; i++) {
			if (i !== 0) {
				html += " / "
			}
			html += HTML
				`<a href="../${boards[i]}/" class="history reload">
					${boards[i]}
				</a>`
		}
		html += HTML
			`] [
			<a class="board-selection bold mono">
				+
			</a>
			]
			</nav>`

		write(() => {
			this.el.innerHTML = html
			document.querySelector("#banner").prepend(this.el)
		})
	}

	private togglePanel(el: Element) {
		if (selectionPanel) {
			selectionPanel.remove()
			selectionPanel = null
		} else {
			selectionPanel = new BoardSelectionPanel(el)
		}
	}
}

// Panel for selecting which boards to display in the top banner
class BoardSelectionPanel extends View<Model> {
	parentEl: Element

	constructor(parentEl: Element) {
		super({})
		this.parentEl = parentEl
		this.render()
		this.onClick({
			"input[name=cancel]": () =>
				this.remove(),
		})
		this.on("submit", e =>
			this.submit(e))
		this.on("input", e => this.search(e), {
			selector: 'input[name=search]'
		})
	}

	// Fetch the board list from the server and render the selection form
	private async render() {
		const frag = makeFrag(await fetchHTML("/forms/boardNavigation"))
		const boards = Array
			.from(frag.querySelectorAll("input[type=checkbox]"))
			.map(b =>
				b.getAttribute("name"))

		// Check all selected boards.
		// Assert all selected boards still exist.If not, deselect them.
		for (let s of selected) {
			if (boards.includes(s)) {
				(frag.querySelector(`input[name=${s}]`) as HTMLInputElement)
					.checked = true
				continue
			}
			selected.delete(s)
			persistSelected()
			navigation.render()
		}

		write(() => {
			this.parentEl.textContent = "-"
			this.el.innerHTML = ""
			this.el.append(frag)
			panel.append(this.el)
		})
	}

	public remove() {
		write(() =>
			this.parentEl.textContent = "+")
		selectionPanel = null
		super.remove()
	}

	// Handle form submission
	private submit(event: Event) {
		event.preventDefault()
		selected.clear()
		for (let el of this.el.querySelectorAll("input[type=checkbox]")) {
			if ((el as HTMLInputElement).checked) {
				selected.add(el.getAttribute("name"))
			}
		}
		persistSelected()
		navigation.render()
		this.remove()
	}

	// Hide board entries that do not match the search field string
	private search(event: Event) {
		const term = (event.target as HTMLInputElement).value.trim(),
			regexp = new RegExp(term, 'i')

		write(() => {
			for (let el of this.el.querySelectorAll("label")) {
				let display: string
				if (regexp.test(el.querySelector("a").textContent)) {
					display = "block"
				} else {
					display = "none"
				}
				el.style.display = display
			}
		})
	}
}

// Write selected boards to localStorage
function persistSelected() {
	const data = JSON.stringify(Array.from(selected))
	localStorage.setItem("selectedBoards", data)
}

// Shift thread to the right, when the side panel is rendered or mutated
function shiftThread() {
	const w = panel.offsetWidth
	if (w === lastPanelWidth) {
		return
	}
	lastPanelWidth = w
	spacer.style.width = w + "px"
}

new MutationObserver(shiftThread).observe(panel, {
	childList: true,
	attributes: true,
	characterData: true,
	subtree: true,
})

// Read selected boards from localStorage
const sel = localStorage.getItem("selectedBoards")
if (sel) {
	for (let b of JSON.parse(sel)) {
		selected.add(b)
	}
}

// Start the module
navigation = new BoardNavigation()
