// Time related aids
import lang from '../lang'
import { pluralize } from "."

export function secondsToTime(s: number): string {
    const divide = [60, 24, 30, 12],
    unit = ['minute', 'hour', 'day', 'month']
    let time = Math.round(s) / 60

    for (let i = 0; i < divide.length; i++) {
        if (time < divide[i]) {
            return pluralize(parseInt(time.toFixed(0), 10), lang.plurals[unit[i]])
        }

        time = Math.round(time / divide[i])
    }

    return pluralize(parseInt(time.toFixed(0), 10), lang.plurals["year"])
}
