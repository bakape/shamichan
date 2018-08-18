// Reload image if error
// Handle event if image error
async function onImageErr(e: Event) {
    const el = (e.target as HTMLImageElement)

    if (el.tagName !== "IMG" || (el.complete && el.naturalWidth !== 0)) {
        return
    }

    const src = el.getAttribute("src")

    if (src.includes("?bs=")) {
        return
    }

    e.stopPropagation()

    for (var i = 0; i < 31; i++) {
        if (el.complete && el.naturalWidth !== 0) {
            break
        }
        
        // Force refresh the cache
        el.setAttribute("src", `${src}?bs=${i}`)
        await new Promise(resolve => setTimeout(resolve, 2000))
    }
}

// Bind listeners
export default () => {
    document.addEventListener("error", onImageErr, { passive: true, capture: true })
}
