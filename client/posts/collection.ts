import { Post } from './model'
import { Model } from "../base"

// Holds a collection of Post models
export default class PostCollection extends Model {
	private models: { [key: string]: Post } = {}
	private static all = new Set<PostCollection>()

	constructor() {
		super()
		PostCollection.all.add(this)
	}

	// Remove a collection from the global registry
	public unregister() {
		PostCollection.all.delete(this)
	}

	// Retrieve a model by ID from all PostCollections in reverse collection
	// creation order
	public static getFromAll(id: number): Post {
		for (let col of [...PostCollection.all].reverse()) {
			const m = col.get(id)
			if (m) {
				return m
			}
		}
		return null
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
