import {accountPannel} from './login'
import {write} from '../render'
import {FormView, FormViewAttrs, FormHandler} from '../forms'

// Generic input form that is embedded into AccountPanel. Takes the parent
// AccountPanel view and function for extracting the form and sending the
// request as parameters.
export default class AccountFormView extends FormView {
	constructor(attrs: FormViewAttrs, handler: FormHandler) {
		super(attrs, handler)
	}

	// Render a form field and embed the input fields inside it. Then append it
	// to the parrent view.
	renderForm(fields: string) {
		super.renderForm(fields)
		accountPannel.hideMenu()
		write(() =>
			accountPannel.el.append(this.el))
	}

	// Unhide the parent AccountPanel, when this view is removed
	remove() {
		super.remove()
		accountPannel.unhideMenu()
	}
}
