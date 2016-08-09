import View from '../view'
import Model from '../model'
import {read, write} from '../render'
import {HTML, makeAttrs} from '../util'
import {fetchBoardList, BoardEntry} from '../fetch'
import Modal from '../modal'
import {ui} from '../lang'
import {formatHeader} from './board'

let boards: BoardEntry[],
	selected: string[],
	navigation: BoardNavigation,
	selectionPanel: BoardSelectionPanel

// View for navigating between boards and selecting w
export default class BoardNavigation extends View<Model> {
	constructor() {
		super({
			id: "board-navigation",
			tag: "nav",
		})
		navigation = this
		const sel = localStorage.getItem("selectedBoards")
		selected = sel ? JSON.parse(sel) : []
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
				`<a href="../${boards[i]}/" class="history">
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
		read(() => {
			const $banner = document.querySelector("#banner")
			write(() =>
				$banner.prepend(this.el))
		})
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
class BoardSelectionPanel extends Modal<Model> {
	parentEl: Element

	constructor(parentEl: Element) {
		super({class: "float-left"})
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

		let boardList = ""
		for (let {id, title} of boards) {
			const attrs: StringMap = {
				type: "checkbox",
				name: id
			}
			if (selected.includes(id)) {
				attrs["checked"] = ""
			}
			boardList += HTML
				`<span class="input-span" data-id="${id}">
					<input ${makeAttrs(attrs)}>
					<label for="${id}">
						${formatHeader(id, title)}
					</label>
					<br>
				</span>`
		}

		const searchAttrs: StringMap = {
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
			document.querySelector("#modal-overlay").append(this.el)
		})
	}

	remove() {
		write(() =>
			this.parentEl.textContent = "+")
		selectionPanel = null
		super.remove()
	}

	// Handle form submition
	submit(event: Event) {
		event.preventDefault()
		selected = []
		for (let el of this.el.querySelectorAll("input[type=checkbox]")) {
			if ((el as HTMLInputElement).checked) {
				selected.push(el.getAttribute("name"))
			}
		}
		localStorage.setItem("selectedBoards", JSON.stringify(selected))
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
				el.style.display =
					matched.includes(el.getAttribute("data-id"))
					? "block"
					: "none"
			}
		})
	}
}
