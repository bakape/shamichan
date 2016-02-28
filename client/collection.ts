import Model from './model'

// Holds a collection of models
export default class Collection {
	models: {[key: string]: Model} = {}

	// Creates a new Collection instance, with optional starting set of models
	constructor(models?: Model[]) {
		if (models) {
			for (let model of models) {
				this.add(model)
			}
		}
	}

	// Add model to collection
	add(model: Model) {
		this.models[model.id] = model
		model.collection = this
	}

	// Remove model from the collection
	remove(model: Model) {
		delete this.models[model.id]
		delete model.collection
	}

	// Remove all models from collection
	clear() {
		for (let id in this.models) {
			delete this.models[id].collection
		}
		this.models = {}
	}

	// Runs the suplied function for each model in the collection
	forEach(fn: (model: Model) => void) {
		for (let id in this.models) {
			fn(this.models[id])
		}
	}
}
