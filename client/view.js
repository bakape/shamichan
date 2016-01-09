import Model from './model'
import {extend} from '../vendor/underscore'

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
        this.model.attach(this)

        // Create element, if none
        if (!this.el) {
            const el = this.el = document.createElement(this.tag || 'div')

            // Set element attributes, if any
            for (let key of ['id', 'class']) {
                if (key in this) {
                    el.setAttribute(key, this[key])
                }
            }
        }
    }

    /**
     * Remove the element from the DOM and detach from its model, allowing the
     * View instance to be garbage collected.
     */
    remove() {
        this.el.remove()
        this.model.detach(this)
        delete this.model
    }

    /**
     * Add selector-specific event listeners to the view
     * @param {string} type - DOM event type
     * @param {string} selector - Selector to match the event.target against
     * @param {string} method - Class method for handling the event
     */
    on(type, selector, method) {
        this.el.addEventListener(type, event => {
            if (event.target.matches(selector)) {
                this[method](event)
            }
        })
    }

    /**
     * Shorthand for adding multiple click event listeners as an object.
     * We use those the most, so nice to have.
     * @param {Object} events - Map of selectors to handlers
     */
    onClick(events) {
        for (let selector in events) {
            this.on('click', selector, events[selector])
        }
    }

    /**
     * Add event listener to view's element, whithout filtering by selector
     * @param {string} type - DOM event type
     * @param {string} method - Class method for handling the event
     */
    onAll(type, method) {
        this.el.addEventListener(type, event => this[method](event))
    }

    /**
     * Add selector-specific event listener, that will execute only once
     * @param {string} type - DOM event type
     * @param {string} selector - Selector to match the event.target against
     * @param {string} method - Class method for handling the event
     */
    once(type, selector, method) {
        this.el.addEventListener(type, event => {
            if (event.target.matches(selector)) {
                this[method](event)
                this.el.removeEventListener(type, this[method])
            }
        })
    }

    /**
     * Add event listener, that will execute only once
     * @param {string} type - DOM event type
     * @param {string} method - Class method for handling the event
     */
    onceAll(type, method) {
        this.el.addEventListener(type, event => {
            this[method](event)
            this.el.removeEventListener(type, this[method])
        })
    }
}
