import AccountFormView, { newRequest, LoginCredentials } from "./common"
import { BoardConfigs } from "../state"
import { makeFrag } from "../util"
import { postJSON, fetchHTML } from "../fetch"
import { loginID, sessionToken } from "./login"
import { write } from "../render"
import View from "../view"

// Board configurations that include a subset not available publically
interface PrivateBoardConfigs extends BoardConfigs {
    banners: string[]
    eightball: string[]
    staff: { [position: string]: string[] }
}

// Request to set the board configs to a new values
interface SettingRequest extends LoginCredentials, PrivateBoardConfigs {
    id: string
}

// Board configuration panel
export default class BoardConfigPanel extends AccountFormView {
    private board: string
    private boardSelector: OwnedBoardSelection

    constructor() {
        const attrs = {
            tag: "form",
            class: "wide-fields",
        }
        super(attrs, () =>
            this.extractRequest()
                .catch(err =>
                    this.renderFormResponse(err)))
        this.boardSelector = new OwnedBoardSelection(this)
        super.render()
    }

    // Render the configuration input elements
    public async renderConfigs(board: string) {
        this.board = board
        const res = await postJSON("/forms/configureBoard", {
            userID: loginID,
            session: sessionToken,
            id: board,
        })
        const frag = makeFrag(await res.text())
        write(() =>
            this.el.append(frag))
    }

    // Extract form data and send a request to apply the new configs
    private async extractRequest() {
        const req = newRequest<SettingRequest>()
        req.id = this.board

        for (let el of this.el.querySelectorAll("input")) {
            const id = el.getAttribute("name")
            switch (el.getAttribute("type")) {
                case "checkbox":
                    req[id] = el.checked
                    break
                case "text":
                    req[id] = el.value
					break
            }
        }

		for (let el of this.el.querySelectorAll("textarea")) {
            const id = el.getAttribute("name")
			req[id] = el.value
			if (id === "eightball") {
				req[id] = req[id].split("\n").slice(0, 100)
            }
		}

        await postJSON("/admin/configureBoard", req)
        this.remove()
    }
}

// Render the <select> for picking the owned board you want to manipulate
class OwnedBoardSelection extends View<null> {
    private parent: BoardConfigPanel

    constructor(parent: BoardConfigPanel) {
        super({ tag: "form" })
        this.parent = parent
        this.on("submit", e =>
            this.onSubmit(e))
        this.render()
    }

    private async render() {
        const html = await fetchHTML(`/forms/ownedBoards/${loginID}`)
        this.el.append(makeFrag(html))
        write(() =>
            this.parent.el.append(this.el))
    }

    private onSubmit(e: Event) {
        e.preventDefault()
        e.stopPropagation()
        const board = (e.target as Element).querySelector("select").value
        this.parent.renderConfigs(board)
        this.remove()
    }
}
