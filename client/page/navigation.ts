import { View } from '../base'
import { HTML, makeFrag, uncachedGET, inputElement } from '../util'

const selected = new Set<string>(),
	panel = document.getElementById("left-panel"),
	spacer = document.getElementById("left-spacer")

let navigation: BoardNavigation,
	selectionPanel: BoardSelectionPanel,
	lastPanelWidth: number

// View for navigating between boards and selecting w
class BoardNavigation extends View<null> {
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
		const catalog = pointToCatalog() ? "catalog" : ""
		for (let i = 0; i < boards.length; i++) {
			if (i !== 0) {
				html += " / "
			}
			html += HTML
				`<a href="../${boards[i]}/${catalog}" class="history reload">
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

		this.el.innerHTML = html
		document.querySelector("#banner").prepend(this.el)
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
class BoardSelectionPanel extends View<null> {
	private parentEl: Element

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
			selector: 'input[name=search]',
			passive: true,
		})
		this.on(
			"change",
			e => {
				const on = (e.target as HTMLInputElement).checked
				this.applyCatalogLinking(on)
			},
			{
				passive: true,
				selector: "input[name=pointToCatalog]",
			},
		)
	}

	// Fetch the board list from the server and render the selection form
	private async render() {
		const r = await uncachedGET("/forms/boardNavigation"),
			text = await r.text()
		if (r.status !== 200) {
			throw text
		}
		const frag = makeFrag(text)
		const boards = Array
			.from(frag.querySelectorAll(".board input"))
			.map(b =>
				b.getAttribute("name"))

		// Check all selected boards.
		// Assert all selected boards still exist.If not, deselect them.
		for (let s of selected) {
			if (boards.includes(s)) {
				inputElement(frag, s).checked = true
				continue
			}
			selected.delete(s)
			persistSelected()
			navigation.render()
		}

		this.el.innerHTML = ""
		this.el.append(frag)

		// Apply and display catalog linking, if any
		if (pointToCatalog()) {
			this.inputElement("pointToCatalog").checked = true
			this.applyCatalogLinking(true)
		}

		this.parentEl.textContent = "-"
		panel.append(this.el)
	}

	public remove() {
		this.parentEl.textContent = "+"
		selectionPanel = null
		super.remove()
	}

	// Handle form submission
	private submit(event: Event) {
		event.preventDefault()
		selected.clear()
		for (let el of this.el.querySelectorAll(".board input")) {
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

		for (let el of this.el.querySelectorAll("label")) {
			let display: string
			if (regexp.test(el.querySelector("a").textContent)) {
				display = "block"
			} else {
				display = "none"
			}
			el.style.display = display
		}
	}

	// Transform links to point to catalog pages and persist
	private applyCatalogLinking(on: boolean) {
		for (let input of this.el.querySelectorAll(".board input")) {
			let url = `/${input.getAttribute("name")}/`
			if (on) {
				url += "catalog"
			}
			(input.nextElementSibling as HTMLAnchorElement).href = url
		}
		localStorage.setItem("pointToCatalog", on.toString())
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

// Returns, if board links should point to catalog pages
function pointToCatalog() {
	return localStorage.getItem("pointToCatalog") === "true"
}

export default () => {
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
}

