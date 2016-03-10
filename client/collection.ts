import Model from './model'

// Holds a collection of models
export default class Collection<M extends Model> {
	models: {[key: number]: M} = {}

	// Creates a new Collection instance, with optional starting set of models
	constructor(models?: M[]) {
		if (models) {
			for (let model of models) {
				this.add(model)
			}
		}
	}

	// Retrieve a model by its ID
	get(id: number): M {
		return this.models[id]
	}

	// Add model to collection
	add(model: M) {
		this.models[model.id] = model
		model.collection = this
	}

	// Remove model from the collection
	remove(model: M) {
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
	forEach(fn: (model: M) => void) {
		for (let id in this.models) {
			fn(this.models[id])
		}
	}
}
