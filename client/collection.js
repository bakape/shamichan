/* @flow */
import Model from './model'

// Holds a collection of models
export default class Collection {
	// Flow can't seem to make sense of a map of imported types.
	// Thus 'any' type.
	models :any = {};

	// Creates a new Collection instance
	constructor(models :Array<Model>) {
		if (models) {
			for (let model of models) {
				this.add(model)
			}
		}
	}

	// Add model to collection
	add(model :Model) {
		this.models[model.id] = model
		model.collection = this
	}

	// Remove model from the collection
	remove(model :Model) {
		delete this.models[model.id]
		delete model.collection
	}

	// Remove all models from collection
	clear() {
		for (let id of this.models) {
			delete this.models[id].collection
		}
		this.models = {}
	}

	/**
	 * Runs the suplied function for each model in the collection
	 * @param {string} method - Method to be called
	 * @param {...*=} args - Arguments to pass
	 */
	forEach(fn :(model :Model) => void) {
		for (let id in this.models) {
			fn(this.models[id])
		}
	}
}
