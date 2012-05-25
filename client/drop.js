(function () {

function drop_shita(e) {
	e.stopPropagation();
	e.preventDefault();
	var files = e.dataTransfer.files;
	if (!files.length)
		return;
	if (!postForm) {
		with_dom(function () {
			if (THREAD)
				open_post_box(THREAD);
			else {
				var $s = $(e.target).closest('section');
				if (!$s.length)
					return;
				open_post_box($s.attr('id'));
			}
		});
	}
	else if (postForm.uploading || postForm.uploaded)
		return;
	if (files.length > 1) {
		postForm.upload_error('Too many files.');
		return;
	}

	var extra = postForm.prep_upload();
	postForm.$imageInput.attr('disabled', true);

	var fd = new FormData();
	fd.append('image', files[0]);
	for (var k in extra)
		fd.append(k, extra[k]);
	/* Can't seem to jQuery this shit */
	var xhr = new XMLHttpRequest();
	xhr.open('POST', 'upload');
	xhr.setRequestHeader('Accept', 'application/json');
	xhr.onreadystatechange = upload_shita;
	xhr.send(fd);
}

function upload_shita() {
	if (this.readyState != 4 || this.status == 202)
		return;
	var err = this.responseText;
	if (this.status != 500 || !err || err.length > 100)
		err = "Couldn't get response.";
	postForm.upload_error(err)
}

function stop_drag(e) {
	e.stopPropagation();
	e.preventDefault();
}

function setup_upload_drop(e) {
	function go(nm, f) { e.addEventListener(nm, f, false); }
	go('dragenter', stop_drag);
	go('dragexit', stop_drag);
	go('dragover', stop_drag);
	go('drop', drop_shita);
}

$(function () {
	setup_upload_drop(document.body);
});

})();
