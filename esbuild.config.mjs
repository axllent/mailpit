import * as esbuild from 'esbuild'
import pluginVue from 'esbuild-plugin-vue-next'
import { sassPlugin } from 'esbuild-sass-plugin'

const doWatch = process.env.WATCH == 'true' ? true : false;
const doMinify = process.env.MINIFY == 'true' ? true : false;

const ctx = await esbuild.context(
    {
        entryPoints: [
            "server/ui-src/app.js",
            "server/ui-src/docs.js"
        ],
        bundle: true,
        minify: doMinify,
        sourcemap: false,
        define: {
            '__VUE_OPTIONS_API__': 'true',
            '__VUE_PROD_DEVTOOLS__': 'false',
            '__VUE_PROD_HYDRATION_MISMATCH_DETAILS__': 'false',
        },
        outdir: "server/ui/dist/",
        plugins: [
            pluginVue(),
            sassPlugin({
                silenceDeprecations: ['import'],
                quietDeps: true,
            })
        ],
        loader: {
            ".svg": "file",
            ".woff": "file",
            ".woff2": "file",
        },
        logLevel: "info"
    }
)

if (doWatch) {
    await ctx.watch()
} else {
    await ctx.rebuild()
    ctx.dispose()
}
