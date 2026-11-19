<script>
import AjaxLoader from "../AjaxLoader.vue";
import CommonMixins from "../../mixins/CommonMixins";
import { domToPng } from "modern-screenshot";
import DOMPurify from "dompurify";
import { screenshotProxyURL } from "../../utils/screenshotProxyURL";

export default {
	components: {
		AjaxLoader,
	},

	mixins: [CommonMixins],

	props: {
		message: {
			type: Object,
			default: () => ({}),
		},
	},

	data() {
		return {
			html: false,
			loading: 0,
		};
	},

	methods: {
		initScreenshot() {
			this.loading = 1;
			const baseUrl = `${location.protocol}//${location.host}/`;
			// absolute proxy URL
			const proxy = new URL(this.resolve("/proxy"), baseUrl);
			const proxyURL = (src) => screenshotProxyURL(proxy, this.message.ID, this.decodeEntities(src));
			const urlRegex = /(url\(('|")?(https?:\/\/[^)'"]+)('|")?\))/gim;

			// remove base tag, if set
			let h = this.message.HTML.replace(/<base .*>/im, "");

			// Outlook hacks - else screenshot returns blank image.
			// Outlook's xmlns:* attributes on the html element are removed by DOMPurify below, so its other
			// attributes such as style (root font-size, used by rem units) and lang are kept.
			h = h.replace(/<o:p><\/o:p>/gm, ""); // remove empty `<o:p></o:p>` tags
			h = h.replace(/<o:/gm, "<"); // replace `<o:p>` tags with `<p>`
			h = h.replace(/<\/o:/gm, "</"); // replace `</o:p>` tags with `</p>`

			// Sanitize HTML before writing to the temporary document.
			// This removes <script>, <noscript>, inline event handlers (on*),
			// SVG <animate>/<set> with xlink:href and other active content
			// that manual tag removal would miss.
			h = DOMPurify.sanitize(h, {
				WHOLE_DOCUMENT: true,
				FORCE_BODY: false,
				ADD_TAGS: ["link", "meta", "o:p", "style"],
				ADD_ATTR: [
					"bordercolor",
					"charset",
					"content",
					"hspace",
					"http-equiv",
					"itemprop",
					"itemscope",
					"itemtype",
					"vertical-align",
					"vlink",
					"vspace",
					"xml:lang",
					"background", // needed for background= URL replacement below
				],
				FORBID_TAGS: ["script", "noscript"],
			});

			// create temporary document to manipulate
			const doc = document.implementation.createHTMLDocument();
			doc.open();
			doc.writeln(h);
			doc.close();

			// replace any url(...) links in <style> blocks
			const styles = doc.getElementsByTagName("style");
			for (const i of styles) {
				i.innerHTML = i.innerHTML.replaceAll(urlRegex, (_match, _url, quote = "", src) => {
					return `url(${quote}${proxyURL(src)}${quote})`;
				});
			}

			// replace stylesheet links with proxy links
			const stylesheets = doc.getElementsByTagName("link");
			for (const i of stylesheets) {
				const src = i.getAttribute("href");
				if (
					src &&
					src.match(/^https?:\/\//i) &&
					src.indexOf(window.location.origin + window.location.pathname) !== 0
				) {
					i.setAttribute("href", proxyURL(src));
				}
			}

			// replace images with proxy links
			const images = doc.getElementsByTagName("img");
			for (const i of images) {
				const src = i.getAttribute("src");
				if (
					src &&
					src.match(/^https?:\/\//i) &&
					src.indexOf(window.location.origin + window.location.pathname) !== 0
				) {
					i.setAttribute("src", proxyURL(src));
				}
			}

			// replace background="" attributes with proxy links
			const backgrounds = doc.querySelectorAll("[background]");
			for (const i of backgrounds) {
				const src = i.getAttribute("background");

				if (
					src &&
					src.match(/^https?:\/\//i) &&
					src.indexOf(window.location.origin + window.location.pathname) !== 0
				) {
					// replace with proxy link
					i.setAttribute("background", proxyURL(src));
				}
			}

			// set html with manipulated document content
			this.html = new XMLSerializer().serializeToString(doc);
		},

		// HTML decode function
		decodeEntities(s) {
			return new DOMParser().parseFromString(s, "text/html").body.textContent;
		},

		doScreenshot() {
			let width = document.getElementById("message-view").getBoundingClientRect().width;

			const prev = document.getElementById("preview-html");
			if (prev && prev.getBoundingClientRect().width) {
				width = prev.getBoundingClientRect().width;
			}

			if (width < 300) {
				width = 300;
			}

			const i = document.getElementById("screenshot-html");

			// set the iframe width
			i.style.width = width + "px";

			const body = i.contentWindow.document.querySelector("body");

			// Add body padding to prevent content touching edge of screenshot.
			body.style.padding = "20px";

			// take screenshot of iframe
			domToPng(body, {
				backgroundColor: "#ffffff",
				height: i.contentWindow.document.body.scrollHeight,
				width,
				// remove the transparent 8px top and left gap from html object (default browser margins).
				style: {
					margin: "0",
				},
			}).then((dataUrl) => {
				const link = document.createElement("a");
				link.download = this.message.ID + ".png";
				link.href = dataUrl;
				link.click();
				this.loading = 0;
				this.html = false;
			});
		},
	},
};
</script>

<template>
	<iframe
		v-if="html"
		id="screenshot-html"
		:srcdoc="html"
		frameborder="0"
		style="position: absolute; margin-left: -100000px"
		@load="doScreenshot"
	>
	</iframe>

	<AjaxLoader :loading="loading" />
</template>
