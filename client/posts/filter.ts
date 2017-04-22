let gallery = false


export function toggleGallery() {
    gallery = !gallery

    document.documentElement.classList.toggle("gallery", gallery)
}