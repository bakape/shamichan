import { ytVids } from "../../state"

// Switch src attribute if in fullscreen mode
// Handle fullscreen activation
function onFullscreen(e: Event) {
    [
        (document as any).fullscreenElement,
        (document as any).webkitFullscreenElement,
        (document as any).mozFullScreenElement,
        (document as any).msFullscreenElement
    ].forEach(async el => {
        if (el) {
            e.stopPropagation()
            e.preventDefault()

            const source = el.querySelector("source")

            if (
                el.hasAttribute("src") ||
                el.hasAttribute("HQ") ||
                !source.getAttribute("src").includes("googlevideo")
            ) {
                return
            }

            const video = ytVids.get(el.getAttribute("poster").split("vi/").pop().split('/').shift())[3],
                oldTime = el.currentTime

            if (!video) {
                console.error("Error: Empty googlevideo URL from ytVids map")
                return
            }

            if (video.includes("mime=video%2Fwebm")) {
                source.setAttribute("type", "video/webm")
            } else {
                source.setAttribute("type", "video/mp4")
            }

            source.setAttribute("src", video)
            el.setAttribute("HQ", true)
            el.load()
            el.currentTime = oldTime
            el.play()
        }
    })
}

// Bind listeners
export default () => {
    ["", "webkit", "moz", "ms"].forEach(
        prefix => document.addEventListener(prefix + "fullscreenchange", onFullscreen)
    );
}
