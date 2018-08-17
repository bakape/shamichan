// Switch src attribute if in fullscreen mode
// Handle fullscreen activation
function onFullscreen(e: Event) {
    [
        document.fullscreenElement,
        document.webkitFullscreenElement,
        document.mozFullScreenElement,
        document.msFullscreenElement
    ].forEach(async el => {
        if (el) {
            e.stopPropagation()
            e.preventDefault()

            if (
                el.hasAttribute("src") ||
                el.hasAttribute("HQ") ||
                !el.querySelector("source").getAttribute("src").includes("googlevideo")
            ) {
                return
            }

            const res = await fetch("/api/youtube-data/" +
                el.getAttribute("poster").split("vi/").pop().split('/').shift()),
            video = (await res.text()).split("\n").pop(),
            oldTime = el.currentTime

            switch (res.status) {
            case 200:
                break
            case 415:
                console.error("Error 415: YouTube video is a livestream")
                return
            case 500:
                console.error("Error 500: YouTube is not available")
                return
            default:
                console.error(`Error ${res.status}: ${res.statusText}`)
                return
            }

            if (!video) {
                console.error("Error: Empty googlevideo URL")
                return
            }

            if (video.includes("mime=video%2Fmp4")) {
                el.querySelector("source").setAttribute("type", "video/mp4")
            }

            el.querySelector("source").setAttribute("src", video)
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
