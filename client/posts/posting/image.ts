// Reload image if error
// Handle event if image error
async function onImageErr(e: Event) {
    const el = (e.target as HTMLImageElement)

    if (el.tagName !== "IMG"
        || (el.complete && el.naturalWidth !== 0)
        || el.getAttribute("data-handling-error")
    ) {
        return
    }
    el.setAttribute("data-handling-error", "1");


    e.stopPropagation()
    e.preventDefault()

    for (var i = 0; i < 30; i++) {
        if (el.complete && el.naturalWidth !== 0) {
            break
        }

        // Force refresh the cache
        el.src = el.src;
        await new Promise(resolve => setTimeout(resolve, 2000))
    }
}

// Bind listeners
export default () => {
    document.addEventListener("error", onImageErr, true)
}
