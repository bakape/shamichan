
const pendingBatches = new Map();

/**
 * Function decorator for vararg functions that allows multiple
 * invocations to be aggregated into a single batch.
 * 
 * Batches are executed every 200ms.
 */
export function timedAggregate<T>(f: (...batch: T[]) => void): (...values: T[]) => void {
    return (...values: T[]) => {
        if (pendingBatches.has(f)) {
            const batch = pendingBatches.get(f);
            batch.push(...values);
            return;
        }
    
        const ary = values;
    
        pendingBatches.set(f, ary);
    
        setTimeout(() => {
            pendingBatches.delete(f);
            f(...ary);
        }, 200);
    }
}