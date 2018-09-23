// Time related aids
import lang from '../lang'

export function secondsToTime(s: number): string {
    const divide = [60, 24, 30, 12];
    const unit = ['minute', 'hour', 'day', 'month'];
    let time = s / 60;

    const format = (key: string) =>
        `${time.toFixed(1)} ${lang.plurals[key][1]}`;

    for (let i = 0; i < divide.length; i++) {
        if (time < divide[i]) {
            return format(unit[i]);
        }
        time /= divide[i];
    }
    return format("year");
}
