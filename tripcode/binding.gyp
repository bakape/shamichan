{
	'targets': [{
		'target_name': 'tripcode',
		"include_dirs" : [
			"<!(node -e \"require('nan')\")"
		],
		'sources': ['tripcode.cc'],
		'link_settings': {
			'conditions': [
				['OS=="linux"', {'libraries': ['-lcrypt']}],
				['OS=="freebsd"', {'libraries': ['-lcrypt']}],
				['OS=="mac"', {'libraries': ['-lcrypto', '/usr/lib/libiconv.dylib']}]
			]
		}
	}]
}
