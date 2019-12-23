const path = require("path");
const CopyPlugin = require("copy-webpack-plugin");
const WasmPackPlugin = require("@wasm-tool/wasm-pack-plugin");

const dist = path.resolve(__dirname, "dist");

module.exports = {
	mode: "production",
	entry: {
		index: "./js/index.js",
	},
	output: {
		path: dist,
		filename: "[name].js",
		publicPath: "/assets/client/",
	},
	devServer: {
		contentBase: dist,
	},
	plugins: [
		new WasmPackPlugin({
			extraArgs: "--no-typescript",
			crateDirectory: __dirname,
		}),
	],
};
