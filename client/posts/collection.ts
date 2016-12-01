import { Post } from './models'

// Holds a collection of Post models
export default class PostCollection {
	public models: { [key: string]: Post } = {}
	public lowestID: number = 0 // Lowest post ID, excluding OP

	// Retrieve a model by its ID
	public get(id: number): Post {
		return this.models[id]
	}

	// Add model to collection
	public add(model: Post) {
		this.models[model.id] = model
		model.collection = this
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
		this.lowestID = 0
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
