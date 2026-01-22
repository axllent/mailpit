<script>
import Attachments from "./MessageAttachments.vue";
import AttachmentDetails from "./AttachmentDetails.vue";
import Headers from "./MessageHeaders.vue";
import HTMLCheck from "./HTMLCheck.vue";
import MessageLinks from "./MessageLinks.vue";
import LinkCheck from "./LinkCheck.vue";
import SpamAssassin from "./SpamAssassin.vue";
import Tags from "bootstrap5-tags";
import { Tooltip } from "bootstrap";
import commonMixins from "../../mixins/CommonMixins";
import { mailbox } from "../../stores/mailbox";
import DOMPurify from "dompurify";
import hljs from "highlight.js/lib/core";
import xml from "highlight.js/lib/languages/xml";

hljs.registerLanguage("html", xml);

export default {
	components: {
		Attachments,
		AttachmentDetails,
		Headers,
		HTMLCheck,
		MessageLinks,
		LinkCheck,
		SpamAssassin,
	},

	mixins: [commonMixins],

	props: {
		message: {
			type: Object,
			required: true,
		},
	},

	emits: ["loadMessages"],

	data() {
		return {
			mailbox,
			srcURI: false,
			iframes: [], // for resizing
			canSaveTags: false, // prevent auto-saving tags on render
			availableTags: [],
			messageTags: [],
			loadHeaders: false,
			htmlScore: false,
			htmlScoreColor: false,
			linkCheckErrors: false,
			spamScore: false,
			spamScoreColor: false,
			showMobileButtons: false,
			showUnsubscribe: false,
			scaleHTMLPreview: "display",
			// keys names match bootstrap icon names
			responsiveSizes: {
				phone: "width: 322px; height: 570px",
				tablet: "width: 768px; height: 1024px",
				display: "width: 100%; height: 100%",
			},
		};
	},

	computed: {
		hasAnyChecksEnabled() {
			return (
				(mailbox.showHTMLCheck && this.message.HTML) ||
				mailbox.showLinkCheck ||
				(mailbox.showSpamCheck && mailbox.uiConfig.SpamAssassin)
			);
		},

		// remove bad HTML, JavaScript, iframes etc
		sanitizedHTML() {
			// set target & rel on all links
			DOMPurify.addHook("afterSanitizeAttributes", (node) => {
				if (
					node.tagName !== "A" ||
					(node.hasAttribute("href") && node.getAttribute("href").substring(0, 1) === "#")
				) {
					return;
				}
				if ("target" in node) {
					node.setAttribute("target", "_blank");
					node.setAttribute("rel", "noopener noreferrer");
				}
				if (!node.hasAttribute("target") && (node.hasAttribute("xlink:href") || node.hasAttribute("href"))) {
					node.setAttribute("xlink:show", "_blank");
				}
			});

			const clean = DOMPurify.sanitize(this.message.HTML, {
				WHOLE_DOCUMENT: true,
				SANITIZE_DOM: false,
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
					"link",
					"vertical-align",
					"vlink",
					"vspace",
					"xml:lang",
				],
				FORBID_ATTR: ["script"], // all JavaScript should be removed
				ALLOW_UNKNOWN_PROTOCOLS: true, // allow link href protocols like myapp:// etc
			});

			// for debugging
			// this.debugDOMPurify(DOMPurify.removed);

			return clean;
		},
	},

	watch: {
		messageTags() {
			if (this.canSaveTags) {
				// save changes to tags
				this.saveTags();
			}
		},

		scaleHTMLPreview(v) {
			if (v === "display") {
				window.setTimeout(() => {
					this.resizeIFrames();
				}, 500);
			}
		},
	},

	mounted() {
		this.canSaveTags = false;
		this.messageTags = this.message.Tags;
		this.renderUI();

		window.addEventListener("resize", this.resizeIFrames);

		const headersTab = document.getElementById("nav-headers-tab");
		headersTab.addEventListener("shown.bs.tab", () => {
			this.loadHeaders = true;
		});

		const rawTab = document.getElementById("nav-raw-tab");
		rawTab.addEventListener("shown.bs.tab", () => {
			this.srcURI = this.resolve("/api/v1/message/" + this.message.ID + "/raw");
			this.resizeIFrames();
		});

		// manually refresh tags
		this.get(this.resolve(`/api/v1/tags`), false, (response) => {
			this.availableTags = response.data;
			this.$nextTick(() => {
				Tags.init("select[multiple]");
				// delay tag change detection to allow Tags to load
				window.setTimeout(() => {
					this.canSaveTags = true;
				}, 200);
			});
		});
	},

	methods: {
		isHTMLTabSelected() {
			this.showMobileButtons = this.$refs.navhtml && this.$refs.navhtml.classList.contains("active");
		},

		renderUI() {
			// activate the first non-disabled tab
			document.querySelector("#nav-tab button:not([disabled])").click();
			document.activeElement.blur(); // blur focus
			document.getElementById("message-view").scrollTop = 0;

			this.isHTMLTabSelected();

			document.querySelectorAll('button[data-bs-toggle="tab"]').forEach((listObj) => {
				listObj.addEventListener("shown.bs.tab", () => {
					this.isHTMLTabSelected();
				});
			});

			const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]');
			[...tooltipTriggerList].map((tooltipTriggerEl) => new Tooltip(tooltipTriggerEl));

			// delay 0.5s until vue has rendered the iframe content
			window.setTimeout(() => {
				const p = document.getElementById("preview-html");
				if (p && typeof p.contentWindow.document.body === "object") {
					try {
						// make links open in new window
						const anchorEls = p.contentWindow.document.body.querySelectorAll("a");
						for (let i = 0; i < anchorEls.length; i++) {
							const anchorEl = anchorEls[i];
							const href = anchorEl.getAttribute("href");

							if (href && href.match(/^https?:\/\//i)) {
								anchorEl.setAttribute("target", "_blank");
							}
						}
					} catch {
						// ignore errors when accessing the iframe content
					}
					this.resizeIFrames();
				}
			}, 500);

			// HTML highlighting
			hljs.highlightAll();
		},

		resizeIframe(el) {
			const i = el.target;
			if (typeof i.contentWindow.document.body.scrollHeight === "number") {
				i.style.height = i.contentWindow.document.body.scrollHeight + 50 + "px";
			}
		},

		resizeIFrames() {
			if (this.scaleHTMLPreview !== "display") {
				return;
			}
			const h = document.getElementById("preview-html");
			if (h) {
				if (typeof h.contentWindow.document.body.scrollHeight === "number") {
					h.style.height = h.contentWindow.document.body.scrollHeight + 50 + "px";
				}
			}
		},

		// set the iframe body & text colors based on current theme
		initRawIframe(el) {
			const bodyStyles = window.getComputedStyle(document.body, null);
			const bg = bodyStyles.getPropertyValue("background-color");
			const txt = bodyStyles.getPropertyValue("color");

			const body = el.target.contentWindow.document.querySelector("body");
			if (body) {
				body.style.color = txt;
				body.style.backgroundColor = bg;
			}

			this.resizeIframe(el);
		},

		// this function is unused but kept here to use for debugging
		debugDOMPurify(removed) {
			if (!removed.length) {
				return;
			}

			const ignoreNodes = ["target", "base", "script", "v:shapes"];

			const d = removed.filter((r) => {
				if (
					typeof r.attribute !== "undefined" &&
					(ignoreNodes.includes(r.attribute.nodeName) || r.attribute.nodeName.startsWith("xmlns:"))
				) {
					return false;
				}
				// inline comments
				if (typeof r.element !== "undefined" && (r.element.nodeType === 8 || r.element.tagName === "SCRIPT")) {
					return false;
				}

				return true;
			});

			if (d.length) {
				console.log(d);
			}
		},

		saveTags() {
			const data = {
				IDs: [this.message.ID],
				Tags: this.messageTags,
			};

			this.put(this.resolve("/api/v1/tags"), data, () => {
				window.scrollInPlace = true;
				this.$emit("loadMessages");
			});
		},

		// Convert plain text to HTML including anchor links
		textToHTML(s) {
			let html = s;

			// RFC2396 appendix E states angle brackets are recommended for text/plain emails to
			// recognize potential spaces in between the URL
			// @see https://www.rfc-editor.org/rfc/rfc2396#appendix-E
			const angleLinks = /<((https?|ftp):\/\/[-\w@:%_+'!.~#?,&//=; ][^>]+)>/gim;
			html = html.replace(angleLinks, "<˱˱˱a href=ˠˠˠ$1ˠˠˠ target=_blank rel=noopener˲˲˲$1˱˱˱/a˲˲˲>");

			// find links without angle brackets, starting with http(s) or ftp
			const regularLinks = /([^ˠ˲]\b)(((https?|ftp):\/\/[-\w@:%_+'!.~#?,&//=;]+))/gim;
			html = html.replace(regularLinks, "$1˱˱˱a href=ˠˠˠ$2ˠˠˠ target=_blank rel=noopener˲˲˲$2˱˱˱/a˲˲˲");

			// plain www links without https?:// prefix
			const shortLinks = /(^|[^/])(www\.[\S]+(\b|$))/gim;
			html = html.replace(
				shortLinks,
				"$1˱˱˱a href=ˠˠˠhttp://$2ˠˠˠ target=ˠˠˠ_blankˠˠˠ rel=ˠˠˠnoopenerˠˠˠ˲˲˲$2˱˱˱/a˲˲˲",
			);

			// escape to HTML & convert <>" characters back
			html = html
				.replace(/&/g, "&amp;")
				.replace(/</g, "&lt;")
				.replace(/>/g, "&gt;")
				.replace(/"/g, "&quot;")
				.replace(/'/g, "&#039;")
				.replace(/˱˱˱/g, "<")
				.replace(/˲˲˲/g, ">")
				.replace(/ˠˠˠ/g, '"');

			return html;
		},
	},
};
</script>

<template>
	<div v-if="message" id="message-view" class="px-2 px-md-0 mh-100">
		<div class="row w-100">
			<div class="col-md">
				<table class="messageHeaders">
					<tbody>
						<tr>
							<th class="small">From</th>
							<td class="privacy">
								<span v-if="message.From">
									<span v-if="message.From.Name" class="text-spaces">
										{{ message.From.Name + " " }}
									</span>
									<span v-if="message.From.Address" class="small">
										&lt;<a :href="searchURI(message.From.Address)" class="text-body">
											{{ message.From.Address }} </a
										>&gt;
									</span>
								</span>
								<span v-else> [ Unknown ] </span>

								<span
									v-if="message.ListUnsubscribe.Header != ''"
									class="small ms-3 link"
									:title="
										showUnsubscribe
											? 'Hide unsubscribe information'
											: 'Show unsubscribe information'
									"
									@click="showUnsubscribe = !showUnsubscribe"
								>
									Unsubscribe
									<i
										class="bi bi bi-info-circle"
										:class="{ 'text-danger': message.ListUnsubscribe.Errors != '' }"
									></i>
								</span>
							</td>
						</tr>
						<tr class="small">
							<th>To</th>
							<td class="privacy">
								<template v-if="message.To && message.To.length">
									<span v-for="(t, i) in message.To" :key="'to_' + i">
										<template v-if="i > 0">, </template>
										<span>
											<span class="text-spaces">{{ t.Name }}</span>
											&lt;<a :href="searchURI(t.Address)" class="text-body"> {{ t.Address }} </a
											>&gt;
										</span>
									</span>
								</template>
								<span v-else class="text-body-secondary">[Undisclosed recipients]</span>
							</td>
						</tr>
						<tr v-if="message.Cc && message.Cc.length" class="small">
							<th>Cc</th>
							<td class="privacy">
								<span v-for="(t, i) in message.Cc" :key="'cc_' + i">
									<template v-if="i > 0">,</template>
									<span class="text-spaces">{{ t.Name }}</span>
									&lt;<a :href="searchURI(t.Address)" class="text-body"> {{ t.Address }} </a>&gt;
								</span>
							</td>
						</tr>
						<tr v-if="message.Bcc && message.Bcc.length" class="small">
							<th>Bcc</th>
							<td class="privacy">
								<span v-for="(t, i) in message.Bcc" :key="'bcc_' + i">
									<template v-if="i > 0">,</template>
									<span class="text-spaces">{{ t.Name }}</span>
									&lt;<a :href="searchURI(t.Address)" class="text-body"> {{ t.Address }} </a>&gt;
								</span>
							</td>
						</tr>
						<tr v-if="message.ReplyTo && message.ReplyTo.length" class="small">
							<th class="text-nowrap">Reply-To</th>
							<td class="privacy text-body-secondary text-break">
								<span v-for="(t, i) in message.ReplyTo" :key="'bcc_' + i">
									<template v-if="i > 0">,</template>
									<span class="text-spaces">{{ t.Name }}</span>
									&lt;<a :href="searchURI(t.Address)" class="text-body-secondary"> {{ t.Address }} </a
									>&gt;
								</span>
							</td>
						</tr>
						<tr
							v-if="message.ReturnPath && message.From && message.ReturnPath != message.From.Address"
							class="small"
						>
							<th class="text-nowrap">Return-Path</th>
							<td class="privacy text-body-secondary text-break">
								&lt;<a :href="searchURI(message.ReturnPath)" class="text-body-secondary">
									{{ message.ReturnPath }} </a
								>&gt;
							</td>
						</tr>
						<tr>
							<th class="small">Subject</th>
							<td>
								<strong v-if="message.Subject != ''" class="text-spaces">{{ message.Subject }}</strong>
								<small v-else class="text-body-secondary">[ no subject ]</small>
							</td>
						</tr>
						<tr class="small">
							<th class="small">Date</th>
							<td>
								{{ messageDate(message.Date) }}
								<small class="ms-2">({{ getFileSize(message.Size) }})</small>
							</td>
						</tr>
						<tr v-if="message.Username" class="small">
							<th class="small">
								Username
								<i
									class="bi bi-exclamation-circle ms-1"
									data-bs-toggle="tooltip"
									data-bs-placement="top"
									data-bs-custom-class="custom-tooltip"
									data-bs-title="The SMTP or send API username the client authenticated with"
								>
								</i>
							</th>
							<td class="small">
								{{ message.Username }}
							</td>
						</tr>
						<tr class="small">
							<th>Tags</th>
							<td>
								<select
									v-model="messageTags"
									class="form-select small tag-selector"
									multiple
									data-full-width="false"
									data-suggestions-threshold="1"
									data-allow-new="true"
									data-clear-end="true"
									data-allow-clear="true"
									data-placeholder="Add tags..."
									data-badge-style="secondary"
									data-regex="^([a-zA-Z0-9\-\ \_\.@]){1,100}$"
									data-separator="|,|"
								>
									<option value="">Type a tag...</option>
									<!-- you need at least one option with the placeholder -->
									<option v-for="t in availableTags" :key="t" :value="t">{{ t }}</option>
								</select>
								<div class="invalid-feedback">Invalid tag name</div>
							</td>
						</tr>

						<tr
							v-if="message.ListUnsubscribe.Header != ''"
							class="small"
							:class="showUnsubscribe ? '' : 'd-none'"
						>
							<th>Unsubscribe</th>
							<td>
								<span v-if="message.ListUnsubscribe.Links.length" class="text-muted small me-2">
									<template v-for="(u, i) in message.ListUnsubscribe.Links">
										<template v-if="i > 0">, </template>
										&lt;{{ u }}&gt;
									</template>
								</span>
								<i
									v-if="message.ListUnsubscribe.HeaderPost != ''"
									class="bi bi-info-circle text-success me-2 link"
									data-bs-toggle="tooltip"
									data-bs-placement="top"
									data-bs-custom-class="custom-tooltip"
									:data-bs-title="'List-Unsubscribe-Post: ' + message.ListUnsubscribe.HeaderPost"
								>
								</i>
								<i
									v-if="message.ListUnsubscribe.Errors != ''"
									class="bi bi-exclamation-circle text-danger link"
									data-bs-toggle="tooltip"
									data-bs-placement="top"
									data-bs-custom-class="custom-tooltip"
									:data-bs-title="message.ListUnsubscribe.Errors"
								>
								</i>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
			<div
				v-if="(message.Attachments && message.Attachments.length) || (message.Inline && message.Inline.length)"
				class="col-md-auto d-none d-md-block text-end mt-md-3"
			>
				<div class="mt-2 mt-md-0">
					<template v-if="message.Attachments.length">
						<span class="badge rounded-pill text-bg-secondary p-2 mb-2" title="Attachments in this message">
							Attachment<span v-if="message.Attachments.length > 1">s</span> ({{
								message.Attachments.length
							}})
						</span>
						<br />
					</template>
					<span
						v-if="message.Inline.length"
						class="badge rounded-pill text-bg-secondary p-2"
						title="Inline images in this message"
					>
						Inline image<span v-if="message.Inline.length > 1">s</span> ({{ message.Inline.length }})
					</span>
				</div>
			</div>
		</div>

		<nav id="nav-tab" class="nav nav-tabs my-3 d-print-none" role="tablist">
			<template v-if="message.HTML">
				<div class="btn-group">
					<button
						id="nav-html-tab"
						ref="navhtml"
						class="nav-link"
						data-bs-toggle="tab"
						data-bs-target="#nav-html"
						type="button"
						role="tab"
						aria-controls="nav-html"
						aria-selected="true"
						@click="resizeIFrames()"
					>
						HTML
					</button>
					<button
						type="button"
						class="nav-link dropdown-toggle dropdown-toggle-split d-sm-none"
						data-bs-toggle="dropdown"
						aria-expanded="false"
						data-bs-reference="parent"
					>
						<span class="visually-hidden">Toggle Dropdown</span>
					</button>
					<div class="dropdown-menu">
						<button
							class="dropdown-item"
							data-bs-toggle="tab"
							data-bs-target="#nav-html-source"
							type="button"
							role="tab"
							aria-controls="nav-html-source"
							aria-selected="false"
						>
							HTML Source
						</button>
					</div>
				</div>
				<button
					id="nav-html-source-tab"
					class="nav-link d-none d-sm-inline"
					data-bs-toggle="tab"
					data-bs-target="#nav-html-source"
					type="button"
					role="tab"
					aria-controls="nav-html-source"
					aria-selected="false"
				>
					HTML <span class="d-sm-none">Src</span><span class="d-none d-sm-inline">Source</span>
				</button>
			</template>

			<button
				id="nav-plain-text-tab"
				class="nav-link"
				data-bs-toggle="tab"
				data-bs-target="#nav-plain-text"
				type="button"
				role="tab"
				aria-controls="nav-plain-text"
				aria-selected="false"
				:class="message.HTML == '' ? 'show' : ''"
			>
				Text
			</button>
			<button
				id="nav-headers-tab"
				class="nav-link"
				data-bs-toggle="tab"
				data-bs-target="#nav-headers"
				type="button"
				role="tab"
				aria-controls="nav-headers"
				aria-selected="false"
			>
				<span class="d-sm-none">Hdrs</span><span class="d-none d-sm-inline">Headers</span>
			</button>
			<button
				id="nav-raw-tab"
				class="nav-link"
				data-bs-toggle="tab"
				data-bs-target="#nav-raw"
				type="button"
				role="tab"
				aria-controls="nav-raw"
				aria-selected="false"
			>
				Raw
			</button>
			<div v-show="hasAnyChecksEnabled" class="dropdown d-xl-none">
				<button class="nav-link dropdown-toggle" type="button" data-bs-toggle="dropdown" aria-expanded="false">
					Checks
				</button>
				<ul class="dropdown-menu checks">
					<li v-if="mailbox.showHTMLCheck && message.HTML != ''">
						<button
							id="nav-html-check-tab"
							class="dropdown-item"
							data-bs-toggle="tab"
							data-bs-target="#nav-html-check"
							type="button"
							role="tab"
							aria-controls="nav-html"
							aria-selected="false"
						>
							HTML Check
							<span
								v-if="htmlScore !== false"
								class="badge rounded-pill p-1 float-end"
								:class="htmlScoreColor"
							>
								<small>{{ Math.floor(htmlScore) }}%</small>
							</span>
						</button>
					</li>
					<li>
						<button
							id="nav-attachments-tab"
							class="dropdown-item"
							data-bs-toggle="tab"
							data-bs-target="#nav-attachments"
							type="button"
							role="tab"
							aria-controls="nav-attachments"
							aria-selected="false"
						>
							Attachments
						</button>
					</li>
					<li>
						<button
							id="nav-links-tab"
							class="dropdown-item"
							data-bs-toggle="tab"
							data-bs-target="#nav-links"
							type="button"
							role="tab"
							aria-controls="nav-links"
							aria-selected="false"
						>
							Links
						</button>
					</li>
					<li v-if="mailbox.showLinkCheck">
						<button
							id="nav-link-check-tab"
							class="dropdown-item"
							data-bs-toggle="tab"
							data-bs-target="#nav-link-check"
							type="button"
							role="tab"
							aria-controls="nav-link-check"
							aria-selected="false"
						>
							Link Check
							<span v-if="linkCheckErrors === 0" class="badge rounded-pill bg-success float-end">
								<small>0</small>
							</span>
							<span v-else-if="linkCheckErrors > 0" class="badge rounded-pill bg-danger float-end">
								<small>{{ formatNumber(linkCheckErrors) }}</small>
							</span>
						</button>
					</li>
					<li v-if="mailbox.showSpamCheck && mailbox.uiConfig.SpamAssassin">
						<button
							id="nav-spam-check-tab"
							class="dropdown-item"
							data-bs-toggle="tab"
							data-bs-target="#nav-spam-check"
							type="button"
							role="tab"
							aria-controls="nav-html"
							aria-selected="false"
						>
							Spam Analysis
							<span
								v-if="spamScore !== false"
								class="badge rounded-pill float-end"
								:class="spamScoreColor"
							>
								<small>{{ spamScore }}</small>
							</span>
						</button>
					</li>
				</ul>
			</div>
			<button
				v-if="mailbox.showHTMLCheck && message.HTML != ''"
				id="nav-html-check-tab"
				class="d-none d-xl-inline-block nav-link position-relative"
				data-bs-toggle="tab"
				data-bs-target="#nav-html-check"
				type="button"
				role="tab"
				aria-controls="nav-html"
				aria-selected="false"
			>
				HTML Check
				<span v-if="htmlScore !== false" class="badge rounded-pill p-1" :class="htmlScoreColor">
					<small>{{ Math.floor(htmlScore) }}%</small>
				</span>
			</button>
			<button
				id="nav-attachments-tab"
				class="d-none d-xl-inline-block nav-link"
				data-bs-toggle="tab"
				data-bs-target="#nav-attachments"
				type="button"
				role="tab"
				aria-controls="nav-attachments"
				aria-selected="false"
			>
				Attachments
			</button>
			<button
				id="nav-links-tab"
				class="d-none d-xl-inline-block nav-link"
				data-bs-toggle="tab"
				data-bs-target="#nav-links"
				type="button"
				role="tab"
				aria-controls="nav-links"
				aria-selected="false"
			>
				Links
			</button>
			<button
				v-if="mailbox.showLinkCheck"
				id="nav-link-check-tab"
				class="d-none d-xl-inline-block nav-link"
				data-bs-toggle="tab"
				data-bs-target="#nav-link-check"
				type="button"
				role="tab"
				aria-controls="nav-link-check"
				aria-selected="false"
			>
				Link Check
				<i v-if="linkCheckErrors === 0" class="bi bi-check-all text-success"></i>
				<span v-else-if="linkCheckErrors > 0" class="badge rounded-pill bg-danger">
					<small>{{ formatNumber(linkCheckErrors) }}</small>
				</span>
			</button>
			<button
				v-if="mailbox.showSpamCheck && mailbox.uiConfig.SpamAssassin"
				id="nav-spam-check-tab"
				class="d-none d-xl-inline-block nav-link position-relative"
				data-bs-toggle="tab"
				data-bs-target="#nav-spam-check"
				type="button"
				role="tab"
				aria-controls="nav-html"
				aria-selected="false"
			>
				Spam Analysis
				<span v-if="spamScore !== false" class="badge rounded-pill" :class="spamScoreColor">
					<small>{{ spamScore }}</small>
				</span>
			</button>

			<div v-if="showMobileButtons" class="d-none d-lg-block ms-auto me-3">
				<template v-for="(_, key) in responsiveSizes" :key="'responsive_' + key">
					<button
						class="btn"
						:disabled="scaleHTMLPreview == key"
						:title="'Switch to ' + key + ' view'"
						@click="scaleHTMLPreview = key"
					>
						<i class="bi" :class="'bi-' + key"></i>
					</button>
				</template>
			</div>
		</nav>

		<div id="nav-tabContent" class="tab-content mb-5">
			<div
				v-if="message.HTML != ''"
				id="nav-html"
				class="tab-pane fade show"
				role="tabpanel"
				aria-labelledby="nav-html-tab"
				tabindex="0"
			>
				<div id="responsive-view" :class="scaleHTMLPreview" :style="responsiveSizes[scaleHTMLPreview]">
					<iframe
						id="preview-html"
						target-blank=""
						class="tab-pane d-block"
						:srcdoc="sanitizedHTML"
						frameborder="0"
						style="width: 100%; height: 100%; background: #fff"
						@load="resizeIframe"
					>
					</iframe>
				</div>
				<Attachments
					v-if="allAttachments(message).length"
					:message="message"
					:attachments="allAttachments(message)"
				>
				</Attachments>
			</div>
			<div
				v-if="message.HTML"
				id="nav-html-source"
				class="tab-pane fade"
				role="tabpanel"
				aria-labelledby="nav-html-source-tab"
				tabindex="0"
			>
				<pre class="language-html"><code class="language-html">{{ message.HTML }}</code></pre>
			</div>
			<div
				id="nav-plain-text"
				class="tab-pane fade"
				role="tabpanel"
				aria-labelledby="nav-plain-text-tab"
				tabindex="0"
				:class="message.HTML == '' ? 'show' : ''"
			>
				<!-- eslint-disable vue/no-v-html -->
				<div class="text-view" v-html="textToHTML(message.Text)"></div>
				<!-- -eslint-disable vue/no-v-html -->
				<Attachments
					v-if="allAttachments(message).length"
					:message="message"
					:attachments="allAttachments(message)"
				>
				</Attachments>
			</div>
			<div id="nav-headers" class="tab-pane fade" role="tabpanel" aria-labelledby="nav-headers-tab" tabindex="0">
				<Headers v-if="loadHeaders" :message="message"></Headers>
			</div>
			<div id="nav-raw" class="tab-pane fade" role="tabpanel" aria-labelledby="nav-raw-tab" tabindex="0">
				<iframe
					v-if="srcURI"
					:src="srcURI"
					frameborder="0"
					style="width: 100%; height: 300px"
					@load="initRawIframe"
				></iframe>
			</div>
			<div
				id="nav-html-check"
				class="tab-pane fade"
				role="tabpanel"
				aria-labelledby="nav-html-check-tab"
				tabindex="0"
			>
				<HTMLCheck
					v-if="mailbox.showHTMLCheck && message.HTML != ''"
					:message="message"
					@set-html-score="(n) => (htmlScore = n)"
					@set-badge-style="(v) => (htmlScoreColor = v)"
				/>
			</div>
			<div
				v-if="mailbox.showSpamCheck && mailbox.uiConfig.SpamAssassin"
				id="nav-spam-check"
				class="tab-pane fade"
				role="tabpanel"
				aria-labelledby="nav-spam-check-tab"
				tabindex="0"
			>
				<SpamAssassin
					:message="message"
					@set-spam-score="(n) => (spamScore = n)"
					@set-badge-style="(v) => (spamScoreColor = v)"
				/>
			</div>
			<div
				id="nav-attachments"
				class="tab-pane fade"
				role="tabpanel"
				aria-labelledby="nav-attachments-tab"
				tabindex="0"
			>
				<AttachmentDetails :message="message" />
			</div>
			<div id="nav-links" class="tab-pane fade" role="tabpanel" aria-labelledby="nav-links-tab" tabindex="0">
				<MessageLinks :message="message" />
			</div>
			<div
				v-if="mailbox.showLinkCheck"
				id="nav-link-check"
				class="tab-pane fade"
				role="tabpanel"
				aria-labelledby="nav-html-check-tab"
				tabindex="0"
			>
				<LinkCheck :message="message" @set-link-errors="(n) => (linkCheckErrors = n)" />
			</div>
		</div>
	</div>
</template>
