const { build } = require('esbuild')
const pluginVue = require('esbuild-plugin-vue-next')
const sassPlugin = require("esbuild-plugin-sass");

const doWatch = process.env.WATCH == 'true' ? true : false;
const doMinify = process.env.MINIFY == 'true' ? true : false;

build({
    entryPoints: ["server/ui-src/app.js"],
    bundle: true,
    watch: doWatch,
    minify: doMinify,
    sourcemap: false,
    outfile: "server/ui/dist/app.js",
    plugins: [pluginVue(), sassPlugin()],
    loader: {
        ".svg": "file",
        ".woff": "file",
        ".woff2": "file",
    },
    logLevel: "info"
})
