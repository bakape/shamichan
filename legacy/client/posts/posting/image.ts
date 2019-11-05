// Reload image if error
// Handle event if image error
function onImageErr(e: Event) {
    const el = (e.target as HTMLImageElement)

    if (el.tagName !== "IMG"
        || (el.complete && el.naturalWidth !== 0)
        || el.getAttribute("data-scheduled-retry")) {
        return
    }

    e.stopPropagation()
    e.preventDefault()

    el.setAttribute("data-scheduled-retry", "1");
    setTimeout(() => retry(el), 2000);
}

// Retry download
function retry(el: HTMLImageElement) {
    if (!document.contains(el) || el.naturalWidth !== 0) {
        el.removeAttribute("data-scheduled-retry");
        return;
    }
    el.src = el.src;
    setTimeout(() => retry(el), 2000);
}

// Bind listeners
export default () => {
    document.addEventListener("error", onImageErr, true)
}
