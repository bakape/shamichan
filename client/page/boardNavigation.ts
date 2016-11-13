import View from '../view'
import Model from '../model'
import { write } from '../render'
import { HTML, makeAttrs } from '../util'
import { fetchBoardList, BoardEntry } from "../fetch"
import { ui } from '../lang'
import { formatHeader } from './board'

const selected = new Set<string>(),
	panel = document.getElementById("left-panel"),
	spacer = document.getElementById("left-spacer")

let boards: BoardEntry[],
	navigation: BoardNavigation,
	selectionPanel: BoardSelectionPanel,
	lastPanelWidth: number

// View for navigating between boards and selecting w
class BoardNavigation extends View<Model> {
	constructor() {
		super({
			id: "board-navigation",
			tag: "nav",
		})

		this.render()
		this.onClick({
			".board-selection": e =>
				this.toggleBoardSelectionPanel(e.target as Element),
		})
	}

	render() {
		let html = "["
		const boards = ["all", ...selected]
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

		write(() =>
			(this.el.innerHTML = html,
				document.querySelector("#banner").prepend(this.el)))
	}

	toggleBoardSelectionPanel(el: Element) {
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
		this.on('input', e => this.search(e), {
			selector: 'input[name=search]'
		})
	}

	// Fetch the board list from the server and render the selection form
	async render() {
		boards = await fetchBoardList()

		// Assert all selected boards still exist. If not, deselect them.
		const boardIDs = boards.map(board =>
			board.id)
		for (let sel of selected) {
			if (boardIDs.indexOf(sel) === -1) {
				selected.delete(sel)
				persistSelected()
				navigation.render()
			}
		}

		let boardList = ""
		for (let {id, title} of boards) {
			const attrs: { [key: string]: string } = {
				type: "checkbox",
				name: id
			}
			if (selected.has(id)) {
				attrs["checked"] = ""
			}
			boardList += HTML
				`<span class="input-span" data-id="${id}">
					<input ${makeAttrs(attrs)}>
					<label for="${id}">
						<a class="history" href="/${id}/">
							${formatHeader(id, title)}
						</a>
					</label>
					<br>
				</span>`
		}

		const searchAttrs = {
			type: "text",
			name: "search",
			placeholder: ui.search,
			class: "full-width",
		}
		const html = HTML
			`<input ${makeAttrs(searchAttrs)}>
			<br>
			<form>
				<input type="submit" value="${ui.apply}">
				<input type="button" name="cancel" value="${ui.cancel}">
				<br>
				${boardList}
			</form>`

		write(() => {
			this.parentEl.textContent = "-"
			this.el.innerHTML = html
			panel.append(this.el)
		})
	}

	remove() {
		write(() =>
			this.parentEl.textContent = "+")
		selectionPanel = null
		super.remove()
	}

	// Handle form submission
	submit(event: Event) {
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
	search(event: Event) {
		const term = (event.target as HTMLInputElement).value.trim(),
			regex = new RegExp(term, 'i'),
			matched: string[] = []
		for (let {id, title} of boards) {
			if (regex.test(id) || regex.test(title) || term === `/${id}/`) {
				matched.push(id)
			}
		}

		write(() => {
			for (let el of this.el.querySelectorAll(`.input-span`)) {
				let display: string
				if (matched.includes(el.getAttribute("data-id"))) {
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
