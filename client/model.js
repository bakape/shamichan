import {extend} from '../vendor/underscore'

/**
 * Generic model class, that all other model classes extend
 */
export default class Model {
    /**
     * Constructs a new model object with the specified attribute object
     * @param {Object} attrs
     */
    constructor(attrs = {}) {
        this.attrs = attrs
        this.views = []
    }

    /**
     * Retrieve a strored value of specific key from the model's attribute
     * object
     * @param {string} key
     * @returns {*}
     */
    get(key) {
        return this.attrs[key]
    }

    /**
     * Set a key to a target value
     * @param {string} key
     * @param {*} val
     */
    set(key, val) {
        this.attrs[key] = val
    }

    /**
     * Extend the model attribute hash, with the suplied object. Shorthand, for
     * setting multiple fields simultaniously.
     * @param {Object} attrs
     */
    setAttrs(attrs) {
        extend(this.attrs, attrs)
    }

    /**
     * Append value to an array strored at the given key. If the array does not
     * exist, it is created.
     * @param {string} key
     * @param {*} val
     */
    append(key, val) {
        if (this.attrs[key]) {
            this.attrs[key].push(val)
        } else {
            this.attrs[key] = [val]
        }
    }

    /**
     * Extend an object at target key. If key does not exist, simply assign the
     * object to the key.
     * @param {string} key
     * @param {Object} object
     */
    extend(key, object) {
        if (this.attrs[key]) {
            extend(this.attrs[key], object)
        } else {
            this.attrs[key] = object
        }
    }

    /**
     * Attach a view to a model. Simply adds the view to the model's view array.
     * Each model's method will then provide individual logic for calling the
     * attached views' methods.
     * @param {View} view
     */
    attach(view) {
        this.views.push(view)
    }

    /**
     * Detach a view from the model. Removes reference, so model and/or view
     * can be garbage collected.
     * @param {View} view
     */
    detach(view) {
        this.views.splice(this.views.indexOf(view), 1)
    }

    /**
     * Remove the model from its collection, if any, and remove all its views
     */
    remove() {
        if (this.collection) {
            this.collection.remove(this)
        }
        for (let view of this.views) {
            view.remove()
        }
    }
}
