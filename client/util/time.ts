// Time related aids
import lang from '../lang'

export function secondsToTime(s: number): string {
    const divide = [60, 60, 24, 30, 12]
    const unit = ['second', 'minute', 'hour', 'day', 'month']
    let time = s

    const format = (key: string) => {
        let tmp = time.toFixed(1)
        let plural = lang.plurals[key][1]

        if (tmp.includes(".0")) {
            tmp = tmp.substr(0, tmp.length - 2)

            if (tmp == '1') {
                plural = lang.plurals[key][0]
            }
        }

        return `${tmp} ${plural}`
    }

    for (let i = 0; i < divide.length; i++) {
        if (time < divide[i]) {
            return format(unit[i])
        }

        time /= divide[i]
    }

    return format("year")
}
