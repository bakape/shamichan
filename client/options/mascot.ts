import { putObj, getObj } from "../db"
import options from "."

type Store = {
	id: string,
	val: Blob,
}

// Listen to  changes in related options, that do not call render() directly
export default () =>
	options.onChange("workModeToggle", render)

export async function store(file: File) {
	const store = {
		id: "mascot",
		val: file,
	}
	await putObj("main", store)
	if (options.mascot) {
		render(store)
	}
}

export async function render(mascot?: Store) {
	const old = document.getElementById("mascot-image")
	if (old) {
		old.remove()
	}
	if (!options.mascot || options.workModeToggle) {
		return
	}

	if (!mascot || !mascot.val) {
		mascot = await getObj<Store>("main", "mascot")
		if (!mascot || !mascot.val) {
			return
		}
	}
	const img = document.createElement("img")
	img.id = "mascot-image"
	img.src = URL.createObjectURL(mascot.val)
	document.body.append(img)
}
