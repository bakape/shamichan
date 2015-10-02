/*
 File drag and drop uploads
 */

const main = require('./main'),
	{etc, state} = main;

function dragonDrop(e) {
	e.stopPropagation();
	e.preventDefault();
	const {files} = e.dataTransfer;
	if (!files.length)
		return;
	let postForm = main.request('postForm');
	if (!postForm) {
		const thread = state.page.get('thread');
		if (thread)
			main.request('openPostBox', thread);
		else {
			const section = e.target.closest('section');
			if (section)
				main.request('openPostBox', section.getAttribute('id'));
		}
	}
	else {
		const attrs = postForm.model.attributes;
		if (attrs.uploading || attrs.uploaded)
			return;
	}

	if (!postForm)
		postForm = main.request('postForm');
	if (files.length > 1) {
		postForm.uploadError('Too many files.');
		return;
	}

	// Drag and drop does not supply a fakepath to file, so we have to use
	// a separate upload form from the postForm one. Meh.
	const extra = postForm.prepareUpload(),
		data = new FormData();
	data.append('image', files[0]);
	for (let key in extra) {
		data.append(key, extra[key]);
	}
	
	const xhr = new XMLHttpRequest();
	xhr.open('POST', etc.uploadURL());
	xhr.setRequestHeader('Accept', 'application/json');
	xhr.onreadystatechange = upload_shita;
	xhr.send(data);

	postForm.notifyUploading();
}

function upload_shita() {
	if (this.readyState != 4 || this.status == 202)
		return;
	const err = this.responseText;
	
	// Everything just fine. Don't need to report.
	if (!/legitimate imager response/.test(err))
		main.request('postForm').uploadError(err);
}

function stop_drag(e) {
	e.stopPropagation();
	e.preventDefault();
}

function setupUploadDrop(el) {
	go('dragenter', stop_drag);
	go('dragexit', stop_drag);
	go('dragover', stop_drag);
	go('drop', dragonDrop);

	function go(name, func) {
		el.addEventListener(name, func, false);
	}
}

if (!main.isMobile)
	main.defer(() => setupUploadDrop(main.$threads[0]));

