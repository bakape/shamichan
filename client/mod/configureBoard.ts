import AccountFormView from "./common"
import { newRequest, extractForm, handle403 } from "./common"
import { makeFrag, extend } from "../util"
import { postJSON } from "../fetch"
import { loginID } from "./login"
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

        const req = newRequest()
        req["id"] = board

        const res = await postJSON("/forms/configureBoard", req)
        switch (res.status) {
            case 200:
                const frag = makeFrag(await res.text())
                write(() =>
                    this.el.append(frag))
                break
            case 403:
                handle403(this)
                break
            default:
                throw await res.text()
        }
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
        const res = await fetch(`/forms/ownedBoards/${loginID}`)
        switch (res.status) {
            case 200:
                this.el.append(makeFrag(await res.text()))
                write(() =>
                    this.parent.el.append(this.el))
                break
            case 403:
                handle403(this.parent)
                break
            default:
                throw await res.text()
        }
    }

    private onSubmit(e: Event) {
        e.preventDefault()
        e.stopPropagation()
        const board = ((e.target as Element)
            .querySelector("select") as HTMLInputElement)
            .value
        this.parent.renderConfigs(board)
        this.remove()
    }
}
