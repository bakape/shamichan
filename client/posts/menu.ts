import View from "../view"
import { Post } from "./models"
import { getModel } from "../state"
import { $threads , write } from "../render"
import { on , outerWidth} from "../util"
import { images } from "../lang"

interface ControlButton extends Element {
	_popup_menu: MenuView
}

// Actions to be performed by the items in the popup menu
const actions: {[key: string]: [string, () => void]} = {
	hide: [images.hide, () =>
		alert("TODO")],
}

// Post header drop down menu
class MenuView extends View<Post> {
	el: HTMLElement
	$parent: ControlButton

	constructor($parent: ControlButton, model: Post) {
		super({
			model,
			tag: "ul",
			class: "popup-menu glass",
		})
		this.$parent = $parent
		$parent._popup_menu = this
		this.render()
		this.on("click", e => this.handleClick(e), {
			passive: true,
		})
	}

	render() {
		for (let key in actions) {
			const $li = document.createElement("li")
			$li.setAttribute("data-id", key)
			$li.textContent = actions[key][0]
			this.el.append($li)
		}

		const {el, $parent} = this
		write(() =>
			$parent.append(el))

		// Calculate position. Can't use CSS translate, because it shifts
		// the background.
		el.style.left =
			el.getBoundingClientRect().left
			- (outerWidth(el) + el.offsetWidth) * 0.6
			+ 'px'
	}

	// Run appropriate handler on click or simply remove the menu
	handleClick(e: Event) {
		actions[(e.target as Element).getAttribute('data-id')][1]()
		this.remove()
	}

	// Also dereference from parent .control element
	remove() {
		this.$parent._popup_menu = null
		super.remove()
	}
}

// Open a popup menu, after clicking on a .control down arrow
function openMenu(e: Event) {
	const parent = (e.target as Element).closest(".control") as ControlButton

	if (parent._popup_menu) {
		return parent._popup_menu.remove()
	}

	const model = getModel(parent)
	if (model) {
		new MenuView(parent, model)
	}
}

export default function bind() {
	on($threads, "click", openMenu, {
		passive: true,
		selector: ".control, .control svg, .control svg path",
	})
}
