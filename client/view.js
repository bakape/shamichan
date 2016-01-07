import Model from './model'
import {extend} from '../vender/underscore'

/**
 * Generic view class, that all over view classes extend
 */
export default class View {
    /**
     * Creates a new View and binds it to the target model. If none, creates a
     * blank model. If no element suplied, creates a new one from tags. Sets
     * some other default variables.
     * @param {Object} args - Attributes to attach to the View instance
     */
    constructor(args) {
        extend(this, args)
        if (!this.model) {
            this.model = new Model()
        }
        model.attach(this)

        // Create element, if none
        if (!this.el) {
            const el = this.el = document.createElement(this.tag || 'div')

            // Set element attributes, if any
            if (this.attrs) {
                for (let key in this.attrs) {
                    el.setAttribute(key, this.attrs[key])
                }
                delete this.attrs
            }
        }

        // Defined in each child class individually
        this.render()
    }

    /**
     * Remove element and unreference view and/or model for garbage collection
     */
    remove() {
        this.model.detach(this)
        this.el.remove()
    }
}
