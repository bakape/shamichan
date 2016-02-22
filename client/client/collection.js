import Model from './model'

// Holds a collection of models
export default class Collection {
	// Creates a new Collection instance
	constructor(models) {
		this.models = {}
		if (models) {
			for (let model of models) {
				this.add(model)
			}
		}
	}

	// Add model to collection
	add(model) {
		this.models[model.id] = model
		model.collection = this
	}

	// Remove model from the collection
	remove(model) {
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

	// Runs the suplied function for each model in the collection
	forEach(fn) {
		for (let id in this.models) {
			fn(this.models[id])
		}
	}
}
