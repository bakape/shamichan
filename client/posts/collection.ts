import {Post} from './models'

// Holds a collection of models
export default class PostCollection{
	models: {[key: string]: Post} = {}

	// Creates a new Collection instance, with optional starting set of models
	constructor(models?: Post[]) {
		if (models) {
			for (let model of models) {
				this.add(model)
			}
		}
	}

	// Retrieve a model by its ID
	get(id: number): Post {
		return this.models[id]
	}

	// Add model to collection
	add(model: Post) {
		this.models[model.id] = model
		model.collection = this
	}

	// Remove model from the collection
	remove(model: Post) {
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
	forEach(fn: (model: Post) => void) {
		for (let id in this.models) {
			fn(this.models[id])
		}
	}
}
