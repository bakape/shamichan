/*
 Detects ES6 features and initialzes the module loader
*/

function initModuleLoader() {
	System.config({
		baseURL: '/ass/js',
		defaultJSExtensions: true,
		map: {
			underscore: 'vendor/underscore-min'
		},

		// Load all client modules as precompiled System.register format
		// modules
		meta: {
			'client/*': {
				format: 'register'
			}
		}
	})
}
