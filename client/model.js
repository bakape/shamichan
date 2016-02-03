/* @flow */
import {extend} from 'underscore'
import View from './view'

type hook = (val :mixed) => void
type hookMap = {[key :string]: hook | Array<hook>}

// Generic model class, that all other model classes extend
export default class Model {
	attrs :Object = {};
	views :Array<View> = [];
	changeHooks :hookMap = {};

	constructor(attrs :?Object) {
		if (attrs) {
			this.attrs = attrs
		}
	}

	// Retrieve a strored value of specific key from the model's attribute
	// object
	get(key :string) :mixed {
		return this.attrs[key]
	}

	// Set a key to a target value
	set(key :string, val :mixed) {
		this.attrs[key] = val
		this.execChangeHooks(key)
	}

	// Extend the model attribute hash, with the suplied object. Shorthand, for
	// setting multiple fields simultaniously.
	setAttrs(attrs :Object) {
		extend(this.attrs, attrs)
		for (let key in attrs) {
			this.execChangeHooks(key)
		}
	}

	// Append value to an array strored at the given key. If the array does not
	// exist, it is created.
	append(key :string, val :mixed) {
		if (this.attrs[key]) {
			this.attrs[key].push(val)
		} else {
			this.attrs[key] = [val]
		}
		this.execChangeHooks(key)
	}

	// Extend an object at target key. If key does not exist, simply assign the
	// object to the key.
	extend(key :string, object :Object) {
		if (this.attrs[key]) {
			extend(this.attrs[key], object)
		} else {
			this.attrs[key] = object
		}
		this.execChangeHooks(key)
	}

	// Attach a view to a model. Simply adds the view to the model's view array.
	// Each model's method will then provide individual logic for calling the
	attach(view :View) {
		this.views.push(view)
	}

	// Detach a view from the model. Removes reference, so model and/or view
	// can be garbage collected.
	detach(view :View) {
		this.views.splice(this.views.indexOf(view), 1)
	}

 	// Remove the model from its collection, if any, and remove all its views
	remove() {
		if (this.collection) {
			this.collection.remove(this)
		}
		for (let view of this.views) {
			view.remove()
		}
	}


 	// Add a function to be executed, when .set(), .setAttrs(), .append() or
	// .extend() modify a key's value.
	onChange(key :string, func :hook) {
		if (this.changeHooks[key]) {
			this.changeHooks[key].push(func)
		} else {
			this.changeHooks[key] = [func]
		}
	}

	// Execute handlers hooked into key change, if any
	// @param {string} key
	execChangeHooks(key :string) {
		if (!this.changeHooks[key]) {
			return
		}
		const val = this.get(key)
		for (let func of this.changeHooks[key]) {
			func(val)
		}
	}
}
