import View from '../view'
import Model from '../model'
import {write} from '../render'
import {HTML} from '../util'
import Modal from '../modal'
import {ui} from '../lang'

let selectionPanel: BoardSelectionPanel

// View for navigating between boards and selecting w
export default class BoardNavigation extends View<Model> {
	selected: string[]

	constructor() {
		super({el: document.querySelector("#board-navigation")})
		const sel = localStorage.getItem("selectedBoards")
		this.selected = sel ? JSON.parse(sel) : []
		this.render()
		this.onClick({
			".board-selection": e =>
				this.toggleBoardSelectionPanel(e.target as Element),
		})
	}

	render() {
		let html = "["
		const boards = ["all", ...this.selected]
		for (let i = 0; i < boards.length; i++) {
			if (i !== 0) {
				html += " / "
			}
			html += `<a href="../${boards[i]}" class="history">${boards[i]}</a>`
		}
		html += HTML
			`] [
			<a class="board-selection bold mono">
				+
			</a>
			]
			</nav>`
		write(() =>
			this.el.innerHTML = html)
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
		super({cls: "float-left"})
		this.parentEl = parentEl
		this.render()
		this.onClick({
			"input[name=cancel]": () =>
				this.remove(),
		})
		this.on("submit", e =>
			this.submit(e))
	}

	render() {
		const html = HTML
			`<form>
				<input type="submit" value="${ui.done}">
				<input type="button" name="cancel" value="${ui.cancel}">
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
		this.remove()
	}
}
