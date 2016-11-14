import { Post, OP } from './models'

// Holds a collection of models
export default class PostCollection {
	public models: { [key: string]: Post } = {}
	public op: OP

	// Creates a new Collection instance, with optional starting set of models
	constructor(models?: Post[]) {
		if (models) {
			for (let model of models) {
				this.add(model)
			}
		}
	}

	// Retrieve a model by its ID
	public get(id: number): Post {
		return this.models[id]
	}

	// Add model to collection
	public add(model: Post) {
		this.models[model.id] = model
		model.collection = this
	}

	// Add the OP of a thread to the collection
	public addOP(model: OP) {
		this.op = model
		this.add(model)
	}

	// Remove model from the collection
	public remove(model: Post) {
		delete this.models[model.id]
		delete model.collection
	}

	// Remove all models from collection
	public clear() {
		for (let id in this.models) {
			delete this.models[id].collection
		}
		this.models = {}
		this.op = null
	}

	// Return weather a post exists in the collection
	public has(id: number): boolean {
		return id in this.models
	}

	// Make collections iterable
	public *[Symbol.iterator](): IterableIterator<Post> {
		yield* Object
			.keys(this.models)
			.map(key =>
				this.models[key])
	}
}
