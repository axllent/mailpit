<script>
import AjaxLoader from "../AjaxLoader.vue";
import CommonMixins from "../../mixins/CommonMixins";
import { domToPng } from "modern-screenshot";

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
			// remove base tag, if set
			let h = this.message.HTML.replace(/<base .*>/im, "");
			const proxy = this.resolve("/proxy");

			// Outlook hacks - else screenshot returns blank image
			h = h.replace(/<html [^>]+>/gim, "<html>"); // remove html attributes
			h = h.replace(/<o:p><\/o:p>/gm, ""); // remove empty `<o:p></o:p>` tags
			h = h.replace(/<o:/gm, "<"); // replace `<o:p>` tags with `<p>`
			h = h.replace(/<\/o:/gm, "</"); // replace `</o:p>` tags with `</p>`

			// update any inline `url(...)` absolute links
			const urlRegex = /(url\(('|")?(https?:\/\/[^)'"]+)('|")?\))/gim;
			h = h.replaceAll(urlRegex, (match, p1, p2, p3) => {
				if (typeof p2 === "string") {
					return `url(${p2}${proxy}?url=` + encodeURIComponent(this.decodeEntities(p3)) + `${p2})`;
				}
				return `url(${proxy}?url=` + encodeURIComponent(this.decodeEntities(p3)) + `)`;
			});

			// create temporary document to manipulate
			const doc = document.implementation.createHTMLDocument();
			doc.open();
			doc.write(h);
			doc.close();

			// remove any <script> tags
			const scripts = doc.getElementsByTagName("script");
			for (const i of scripts) {
				i.parentNode.removeChild(i);
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
					i.setAttribute("href", `${proxy}?url=` + encodeURIComponent(this.decodeEntities(src)));
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
					i.setAttribute("src", `${proxy}?url=` + encodeURIComponent(this.decodeEntities(src)));
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
					i.setAttribute("background", `${proxy}?url=` + encodeURIComponent(this.decodeEntities(src)));
				}
			}

			// set html with manipulated document content
			this.html = new XMLSerializer().serializeToString(doc);
		},

		// HTML decode function
		decodeEntities(s) {
			const e = document.createElement("div");
			e.innerHTML = s;
			const str = e.textContent;
			e.textContent = "";
			return str;
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

			// take screenshot of iframe
			domToPng(body, {
				backgroundColor: "#ffffff",
				height: i.contentWindow.document.body.scrollHeight + 20,
				width,
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
