import { Post, OP } from './models'

// Holds a collection of models
export default class PostCollection {
	models: { [key: string]: Post } = {}
	op: OP

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

	// Add the OP of a thread to the collection
	addOP(model: OP) {
		this.op = model
		this.add(model)
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
		this.op = null
	}

	// Return weather a post exists in the collection
	has(id: number): boolean {
		return id in this.models
	}

	// Make collections itterable
	*[Symbol.iterator](): IterableIterator<Post> {
		yield* Object
			.keys(this.models)
			.map(key =>
				this.models[key])
	}
}
