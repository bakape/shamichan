import { SelectedBoardForm } from "./common"
import { newRequest, extractForm, handle403 } from "./common"
import { makeFrag, extend } from "../util"
import { postJSON } from "../fetch"
import { write } from "../render"

// Board configuration panel
export default class BoardConfigPanel extends SelectedBoardForm {
    constructor() {
        super({ class: "wide-fields" })
    }

    // Render the configuration input elements
    public async renderNext(board: string) {
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
    protected send() {
        const req = newRequest()
        req["id"] = this.board
        extend(req, extractForm(this.el))

        // TODO: Some kind of form for inputting arrays
        req["eightball"] = req["eightball"].split("\n").slice(0, 100)

        this.postResponse("/admin/configureBoard", req)
    }
}
