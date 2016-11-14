// import AccountFormView, { newRequest, LoginCredentials } from "./common"
// import { BoardConfigs } from "../state"
// import { InputSpec, renderInput, inputType } from '../forms'
// import { admin as lang } from '../lang'
// import { table, makeFrag, makeEl, on } from "../util"
// import { fetchJSON, fetchBoardList, postJSON } from "../fetch"
// import { loginID, sessionToken } from "./login"
// import { write } from "../render"
// import { formatHeader } from "../page/board"

// // Board configurations that include a subset not available publically
// interface PrivateBoardConfigs extends BoardConfigs {
// 	banners: string[]
// 	eightball: string[]
// 	staff: { [position: string]: string[] }
// }

// // Request to set the board configs to a new values
// interface SettingRequest extends LoginCredentials, PrivateBoardConfigs {
// 	id: string
// }

// const specs: InputSpec[] = [
// 	{
// 		name: "readOnly",
// 		type: inputType.boolean,
// 	},
// 	{
// 		name: "textOnly",
// 		type: inputType.boolean,
// 	},
// 	{
// 		name: "forcedAnon",
// 		type: inputType.boolean,
// 	},
// 	{
// 		name: "hashCommands",
// 		type: inputType.boolean,
// 	},
// 	{
// 		name: "spoilers",
// 		type: inputType.boolean,
// 	},
// 	{
// 		name: "codeTags",
// 		type: inputType.boolean,
// 	},

// 	// TODO: Spoiler upload

// 	{
// 		name: "title",
// 		type: inputType.string,
// 		maxLength: 100,
// 	},
// 	{
// 		name: "notice",
// 		type: inputType.multiline,
// 		rows: 5,
// 		maxLength: 500,
// 	},
// 	{
// 		name: "rules",
// 		type: inputType.multiline,
// 		rows: 5,
// 		maxLength: 5000,
// 	},
// 	{
// 		name: "eightball",
// 		type: inputType.multiline,
// 		rows: 5,
// 		maxLength: 2000,
// 	}

// 	// TODO: Banner upload
// 	// TODO: Staff configuration

// ]

// // Board configuration panel
// export default class BoardConfigPanel extends AccountFormView {
// 	board: string

// 	constructor() {
// 		const attrs = {
// 			class: "wide-fields",
// 			noCaptcha: true,
// 		}
// 		super(attrs, () =>
// 			this.extractRequest()
// 				.catch(err =>
// 					this.renderFormResponse(err)))
// 		this.renderSelection()
// 			.catch(err =>
// 				this.renderFormResponse(err))
// 	}

// 	// Render the radio element for picking the board you want to configure
// 	async renderSelection() {
// 		const path = `/json/positions/owners/${loginID}`,
// 			fBoards = fetchJSON<string[]>(path),
// 			fBoardList = fetchBoardList(),
// 			boards = (await fBoards).sort()
// 		let boardList = await fBoardList

// 		if (!boards.length) {
// 			this.noSubmit = true
// 			const html = lang["ownNoBoards"] as string + "<br><br>"
// 			this.renderForm(makeFrag(html))
// 			return
// 		}

// 		// Filter boards we do not own from the list
// 		boardList = boardList.filter(board =>
// 			boards.indexOf(board.id) > -1)

// 		let html = "<span>"
// 		for (let board of boardList) {
// 			const header = formatHeader(board.id, board.title)
// 			html += ` <a data-value="${board.id}"}>${header}</a><br>`
// 		}
// 		html += "<br></span>"

// 		const board = makeEl(html)
// 		const handler = (event: MouseEvent) => {
// 			board.remove()
// 			this.el.querySelector("input[type=submit]").style.display = ""
// 			const val = (event.target as Element).getAttribute("data-value")
// 			this.renderConfigs(val)
// 		}
// 		on(board, "click", handler, {
// 			capture: true,
// 			selector: "a",
// 		})

// 		this.renderForm(board)
// 		this.el.querySelector("input[type=submit]").style.display = "none"
// 	}

// 	// Render the configuration input elements
// 	async renderConfigs(board: string) {
// 		this.board = board
// 		const res = await postJSON("/admin/boardConfig", {
// 			userID: loginID,
// 			session: sessionToken,
// 			id: board,
// 		})
// 		const conf: PrivateBoardConfigs = await res.json()
// 		conf.eightball = conf.eightball.join("\n") as any

// 		const html = table(specs, spec => {
// 			[spec.label, spec.tooltip] = lang[spec.name]
// 			spec.value = conf[spec.name]
// 			return renderInput(spec)
// 		})
// 		write(() =>
// 			this.el.prepend(makeFrag(html)))
// 	}

// 	// Extract form data and send a request to apply the new configs
// 	async extractRequest() {
// 		const req = newRequest<SettingRequest>()
// 		req.id = this.board
// 		for (let {name, type} of specs) {
// 			const el = this.el
// 				.querySelector(`[name=${name}]`) as HTMLInputElement
// 			switch (type) {
// 				case inputType.boolean:
// 					req[name] = el.checked
// 					break
// 				default:
// 					req[name] = el.value
// 			}
// 		}
// 		req.eightball = (req.eightball as any).split("\n").slice(0, 100)

// 		await postJSON("/admin/configureBoard", req)
// 		this.remove()
// 	}
// }
