
/**
 * Holds a collection of models
 */
export default class Collection {
    /**
     * Creates a new Collection instance
     * @param {Model[]=} models - Suplied array of models
     */
    constructor(models) {
        this.models = {}
        if (models) {
            for (let model of models) {
                this.add(model)
            }
        }
    }

    /**
     * Add model to collection
     * @param {Model} model
     */
    add(model) {
        this.models[model.id] = model
        model.collection = this
    }

    /**
     * Remove model from the collection
     * @param {Model} model
     */
    remove(model) {
        delete this.models[model.id]
        delete model.collection
    }

    /**
     * Remove all models from collection
     */
    clear() {
        for (let id of this.models) {
            delete this.models[id].collection
        }
        this.models = {}
    }

    /**
     * Runs the suplied model method for each model in the collection
     * @param {string} method - Method to be called
     * @param {...*=} args - Arguments to pass
     */
    forEach(method, ...args) {
        for (let id in this.models) {
            this.models[id][method](...args)
        }
    }
}
