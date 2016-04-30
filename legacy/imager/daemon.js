/*
 Image and video upload processing
*/

class ImageUpload {
	verify_webm(err, info) {
		if (err)
			return this.failure(Muggle(this.lang[err] || err));
		this.db.track_temporary(info.still_path, err => {
			if (err)
				winston.warn("Tracking error: " + err);

			if (info.audio && !config.WEBM_AUDIO)
				return this.failure(Muggle(this.lang.audio_kinshi));

			// pretend it's a PNG for the next steps
			const {image} = this;
			image.video = image.path;
			image.path = info.still_path;
			image.ext = '.png';
			for (let prop of ['audio', 'length', 'mp3']) {
				if (info[prop])
					image[prop] = info[prop];
			}

			this.verify_image();
		});
	}
}
