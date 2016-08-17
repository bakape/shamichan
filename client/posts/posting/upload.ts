import {posts as lang} from '../../lang'
import {HTML, commaList, load, makeAttrs} from '../../util'
import {read, write} from '../../render'

// Uploaded file data to be embeded in thread and reply creation or appendage
// requests
export interface FileData {
	imageToken: string
	imageName: string
	spoiler: boolean
	[index: string]: any
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
			this.$uploadInput = this
				.el.querySelector("input[name=image]") as HTMLInputElement
		})

		const attrs: StringMap = {
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
	// Assigns the file data to the passed request object.
	async uploadFile(req: FileData): Promise<boolean> {
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
			return false
		}
		req.imageName = file.name
		req.imageToken = xhr.response
		req.spoiler =
			(this.el.querySelector("input[name=spoiler]") as HTMLInputElement)
			.checked
		return true
	}

	renderProgress({total, loaded}: ProgressEvent) {
		write(() =>
			this.$uploadStatus.textContent =
			`${formatProgress(loaded, total)} ${lang.uploadProgress}`)
	}
}

const formatProgress = (done: number, total: number): string =>
	Math.floor(done / total * 100) + "%"
