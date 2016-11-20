import AccountFormView from "./common"
import { newRequest, extractForm } from "./common"
import { makeFrag, extend } from "../util"
import { postJSON, fetchHTML } from "../fetch"
import { loginID, sessionToken } from "./login"
import { write } from "../render"
import View from "../view"

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
        const req = newRequest()
        req["id"] = this.board
        extend(req, extractForm(this.el))

        // TODO: Some kind of form for inputting arrays
        req["eightball"] = req["eightball"].split("\n").slice(0, 100)

        this.postResponse("/admin/configureBoard", req)
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
        const [html, err] = await fetchHTML(`/forms/ownedBoards/${loginID}`)
        if (err) {
            throw err
        }
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
