import {posts as lang} from '../../lang'
import {HTML, commaList, load, setAttrs, makeFrag} from '../../util'
import {write} from '../../render'

// Uploaded file data to be embeded in thread and reply creation or appendage
// requests
export type FileData = {
	token: string
	name: string
	spoiler?: boolean
}

const acceptedFormats = commaList([
	"image/png", "image/gif", "image/jpeg", "video/webm", "video/mp4",
	"video/ogg", "application/pdf", "audio/mp3", "image/svg"
])

// Mixin for handling file uploads
export default class UploadForm {
	el: Element
	$spoiler: HTMLSpanElement
	$uploadStatus: Element
	$uploadInput: HTMLInputElement

	// Initialize the mixin by rendering and assigning the various upload form
	// elements
	renderUploadForm() {
		this.$uploadInput = document.createElement("input")
		setAttrs(this.$uploadInput, {
			type: "file",
			name: "image",
			accept: acceptedFormats,
			required: "",
		})

		this.$spoiler = document.createElement("span")
		const html = HTML
			`<input type="checkbox" name="spoiler">
			<label for="spoiler" class="spoiler">
				${lang.spoiler}
			</label>`
		this.$spoiler.append(makeFrag(html))

		this.$uploadStatus = document.createElement("strong")
		this.$uploadStatus.setAttribute("class", "upload-status")
	}

	// Read the file from $uploadInput and send as a POST request to the server.
	// Returns image request data, if upload succeeded.
	async uploadFile(): Promise<FileData> {
		const formData = new FormData(),
			file = this.$uploadInput.files[0]
		formData.append("image", file)

		// Not using fetch, because no ProgressEvent support
		const xhr = new XMLHttpRequest()
		xhr.open("POST", "/upload")
		xhr.upload.onprogress = e =>
			this.renderProgress(e)
		xhr.send(formData)
		await load(xhr)

		if (xhr.status !== 200) {
			write(() =>
				this.$uploadStatus.textContent = xhr.response)
			return null
		}

		const img: FileData = {
			name: file.name,
			token: xhr.response,
		}
		const spoiler =
			(this.el.querySelector("input[name=spoiler]") as HTMLInputElement)
			.checked
		if (spoiler) {
			img.spoiler = true
		}
		return img
	}

	renderProgress({total, loaded}: ProgressEvent) {
		let s: string
		if (loaded === total) {
			s = lang.thumbnailing
		} else {
			s = `${Math.floor(loaded / total * 100)}% ${lang.uploadProgress}`
		}
		write(() =>
			this.$uploadStatus.textContent = s)
	}
}
