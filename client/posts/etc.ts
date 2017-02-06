import { on } from "../util"

// Toggle spoiler revealing on click
function toggleSpoiler(event: Event) {
	(event.target as Element).classList.toggle("reveal")
}

export default () =>
	on(
		document.getElementById("threads"),
		"click",
		toggleSpoiler,
		{ selector: "del" },
	)
