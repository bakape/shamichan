import {posts as lang} from '../../lang'
import {HTML, commaList, load, makeAttrs} from '../../util'
import {read, write} from '../../render'

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
	$uploadStatus: Element
	$uploadInput: HTMLInputElement

	// Return the HTML of the file upload elements and intialize the mixin
	renderUploadForm() {
		read(() => {
			this.$uploadStatus = this.el.querySelector(".upload-status")
			this.$uploadInput =
				this.el
				.querySelector("input[name=image]") as HTMLInputElement
		})

		const attrs = {
			type: "file",
			name: "image",
			accept: acceptedFormats,
			required: "",
		}
		return HTML
			`<span class="upload-container">
				<input type="checkbox" name="spoiler">
				<label for="spoiler" class="spoiler">
					${lang.spoiler}
				</label>
				<strong class="upload-status"></strong>
				<br>
				<input ${makeAttrs(attrs)}>
			</span>`
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
		write(() =>
			this.$uploadStatus.textContent =
			`${formatProgress(loaded, total)} ${lang.uploadProgress}`)
	}
}

function formatProgress(done: number, total: number): string {
	return Math.floor(done / total * 100) + "%"
}
