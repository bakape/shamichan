// Time related aids
import lang from '../lang'
import { pad } from './index'

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

export function secondsToTimeExact(s: number): string {
	let time: string
	const hours = Math.floor(s/3600),
		minutes = Math.floor((s-hours*3600)/60),
		seconds = Math.round((s-hours*3600-minutes*60))
	time = hours + ":" + minutes + ":" + seconds
	if (hours) {
		time = hours + ":" + pad(minutes) + ":" + pad(seconds)
	} else if(minutes) {
		time = minutes + ":" + pad(seconds)
	} else {
		time = "00:" + pad(seconds)
	}
	return time
}
