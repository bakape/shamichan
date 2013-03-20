{
	'targets': [{
		'target_name': 'tripcode',
		'sources': ['tripcode.cc'],
		'link_settings': {
			'conditions': [
				['OS=="linux"', {'libraries': ['-lcrypt']}],
				['OS=="freebsd"', {'libraries': ['-lcrypt']}],
				['OS=="mac"', {'libraries': ['-lcrypto']}]
			]
		}
	}]
}
