<script>
import Attachments from './Attachments.vue'
import Headers from './Headers.vue'
import HTMLCheck from './HTMLCheck.vue'
import LinkCheck from './LinkCheck.vue'
import SpamAssassin from './SpamAssassin.vue'
import Prism from 'prismjs'
import Tags from 'bootstrap5-tags'
import { Tooltip } from 'bootstrap'
import commonMixins from '../../mixins/CommonMixins'
import { mailbox } from '../../stores/mailbox'
import DOMPurify from 'dompurify'

export default {
	props: {
		message: Object,
	},

	components: {
		Attachments,
		Headers,
		HTMLCheck,
		LinkCheck,
		SpamAssassin,
	},

	mixins: [commonMixins],

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
			scaleHTMLPreview: 'display',
			// keys names match bootstrap icon names 
			responsiveSizes: {
				phone: 'width: 322px; height: 570px',
				tablet: 'width: 768px; height: 1024px',
				display: 'width: 100%; height: 100%',
			},
		}
	},

	watch: {
		messageTags() {
			if (this.canSaveTags) {
				// save changes to tags
				this.saveTags()
			}
		},

		scaleHTMLPreview(v) {
			if (v == 'display') {
				window.setTimeout(() => {
					this.resizeIFrames()
				}, 500)
			}
		}
	},

	computed: {
		hasAnyChecksEnabled() {
			return (mailbox.showHTMLCheck && this.message.HTML)
				|| mailbox.showLinkCheck
				|| (mailbox.showSpamCheck && mailbox.uiConfig.SpamAssassin)
		},

		// remove bad HTML, JavaScript, iframes etc
		sanitizedHTML() {
			// set target & rel on all links
			DOMPurify.addHook('afterSanitizeAttributes', (node) => {
				if (node.tagName != 'A' || (node.hasAttribute('href') && node.getAttribute('href').substring(0, 1) == '#')) {
					return
				}
				if ('target' in node) {
					node.setAttribute('target', '_blank');
					node.setAttribute('rel', 'noopener noreferrer');
				}
				if (!node.hasAttribute('target') && (node.hasAttribute('xlink:href') || node.hasAttribute('href'))) {
					node.setAttribute('xlink:show', '_blank');
				}
			});

			const clean = DOMPurify.sanitize(
				this.message.HTML,
				{
					WHOLE_DOCUMENT: true,
					SANITIZE_DOM: false,
					ADD_TAGS: [
						'link',
						'meta',
						'o:p',
						'style',
					],
					ADD_ATTR: [
						'bordercolor',
						'charset',
						'content',
						'hspace',
						'http-equiv',
						'itemprop',
						'itemscope',
						'itemtype',
						'link',
						'vertical-align',
						'vlink',
						'vspace',
						'xml:lang',
					],
					FORBID_ATTR: ['script'],
				}
			)

			// for debugging
			// this.debugDOMPurify(DOMPurify.removed)

			return clean
		}
	},

	mounted() {
		this.canSaveTags = false
		this.messageTags = this.message.Tags
		this.renderUI()

		window.addEventListener("resize", this.resizeIFrames)

		let headersTab = document.getElementById('nav-headers-tab')
		headersTab.addEventListener('shown.bs.tab', (event) => {
			this.loadHeaders = true
		})

		let rawTab = document.getElementById('nav-raw-tab')
		rawTab.addEventListener('shown.bs.tab', (event) => {
			this.srcURI = this.resolve('/api/v1/message/' + this.message.ID + '/raw')
			this.resizeIFrames()
		})

		// manually refresh tags
		this.get(this.resolve(`/api/v1/tags`), false, (response) => {
			this.availableTags = response.data
			this.$nextTick(() => {
				Tags.init('select[multiple]')
				// delay tag change detection to allow Tags to load
				window.setTimeout(() => {
					this.canSaveTags = true
				}, 200)
			})
		})
	},

	methods: {
		isHTMLTabSelected() {
			this.showMobileButtons = this.$refs.navhtml
				&& this.$refs.navhtml.classList.contains('active')
		},

		renderUI() {
			// activate the first non-disabled tab
			document.querySelector('#nav-tab button:not([disabled])').click()
			document.activeElement.blur() // blur focus
			document.getElementById('message-view').scrollTop = 0

			this.isHTMLTabSelected()

			document.querySelectorAll('button[data-bs-toggle="tab"]').forEach((listObj) => {
				listObj.addEventListener('shown.bs.tab', (event) => {
					this.isHTMLTabSelected()
				})
			})

			const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]');
			[...tooltipTriggerList].map(tooltipTriggerEl => new Tooltip(tooltipTriggerEl))

			// delay 0.5s until vue has rendered the iframe content
			window.setTimeout(() => {
				let p = document.getElementById('preview-html')
				if (p && typeof p.contentWindow.document.body == 'object') {
					try {
						// make links open in new window
						let anchorEls = p.contentWindow.document.body.querySelectorAll('a')
						for (var i = 0; i < anchorEls.length; i++) {
							let anchorEl = anchorEls[i]
							let href = anchorEl.getAttribute('href')

							if (href && href.match(/^https?:\/\//i)) {
								anchorEl.setAttribute('target', '_blank')
							}
						}
					} catch (error) { }
					this.resizeIFrames()
				}
			}, 500)

			// html highlighting
			window.Prism = window.Prism || {}
			window.Prism.manual = true
			Prism.highlightAll()
		},

		resizeIframe(el) {
			let i = el.target
			if (typeof i.contentWindow.document.body.scrollHeight == 'number') {
				i.style.height = i.contentWindow.document.body.scrollHeight + 50 + 'px'
			}
		},

		resizeIFrames() {
			if (this.scaleHTMLPreview != 'display') {
				return
			}
			let h = document.getElementById('preview-html')
			if (h) {
				if (typeof h.contentWindow.document.body.scrollHeight == 'number') {
					h.style.height = h.contentWindow.document.body.scrollHeight + 50 + 'px'
				}
			}

		},

		// set the iframe body & text colors based on current theme
		initRawIframe(el) {
			let bodyStyles = window.getComputedStyle(document.body, null)
			let bg = bodyStyles.getPropertyValue('background-color')
			let txt = bodyStyles.getPropertyValue('color')

			let body = el.target.contentWindow.document.querySelector('body')
			if (body) {
				body.style.color = txt
				body.style.backgroundColor = bg
			}

			this.resizeIframe(el)
		},

		// this function is unused but kept here to use for debugging
		debugDOMPurify(removed) {
			if (!removed.length) {
				return
			}

			const ignoreNodes = ['target', 'base', 'script', 'v:shapes']

			let d = removed.filter((r) => {
				if (typeof r.attribute != 'undefined' &&
					(ignoreNodes.includes(r.attribute.nodeName) || r.attribute.nodeName.startsWith('xmlns:'))
				) {
					return false
				}
				// inline comments
				if (typeof r.element != 'undefined' && (r.element.nodeType == 8 || r.element.tagName == 'SCRIPT')) {
					return false
				}

				return true
			})

			if (d.length) {
				console.log(d)
			}
		},

		saveTags() {
			var data = {
				IDs: [this.message.ID],
				Tags: this.messageTags
			}

			this.put(this.resolve('/api/v1/tags'), data, (response) => {
				window.scrollInPlace = true
				this.$emit('loadMessages')
			})
		},

		// Convert plain text to HTML including anchor links
		textToHTML(s) {
			let html = s

			// full links with http(s)
			let re = /(\b(https?|ftp):\/\/[\-\w@:%_\+'!.~#?,&\/\/=;]+)/gim
			html = html.replace(re, '˱˱˱a href=ˠˠˠ$&ˠˠˠ target=_blank rel=noopener˲˲˲$&˱˱˱/a˲˲˲')

			// plain www links without https?:// prefix
			let re2 = /(^|[^\/])(www\.[\S]+(\b|$))/gim
			html = html.replace(re2, '$1˱˱˱a href=ˠˠˠhttp://$2ˠˠˠ target=ˠˠˠ_blankˠˠˠ rel=ˠˠˠnoopenerˠˠˠ˲˲˲$2˱˱˱/a˲˲˲')

			// escape to HTML & convert <>" back
			html = html
				.replace(/&/g, "&amp;")
				.replace(/</g, "&lt;")
				.replace(/>/g, "&gt;")
				.replace(/"/g, "&quot;")
				.replace(/'/g, "&#039;")
				.replace(/˱˱˱/g, '<')
				.replace(/˲˲˲/g, '>')
				.replace(/ˠˠˠ/g, '"')

			return html
		},
	}
}
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
									<span v-if="message.From.Name" class="text-spaces">{{ message.From.Name + " "
										}}</span>
									<span v-if="message.From.Address" class="small">
										&lt;<a :href="searchURI(message.From.Address)" class="text-body">
											{{ message.From.Address }}
										</a>&gt;
									</span>
								</span>
								<span v-else>
									[ Unknown ]
								</span>

								<span v-if="message.ListUnsubscribe.Header != ''" class="small ms-3 link"
									:title="showUnsubscribe ? 'Hide unsubscribe information' : 'Show unsubscribe information'"
									@click="showUnsubscribe = !showUnsubscribe">
									Unsubscribe
									<i class="bi bi bi-info-circle"
										:class="{ 'text-danger': message.ListUnsubscribe.Errors != '' }"></i>
								</span>
							</td>
						</tr>
						<tr class="small">
							<th>To</th>
							<td class="privacy">
								<span v-if="message.To && message.To.length" v-for="(t, i) in message.To">
									<template v-if="i > 0">, </template>
									<span>
										<span class="text-spaces">{{ t.Name }}</span>
										&lt;<a :href="searchURI(t.Address)" class="text-body">
											{{ t.Address }}
										</a>&gt;
									</span>
								</span>
								<span v-else class="text-body-secondary">[Undisclosed recipients]</span>
							</td>
						</tr>
						<tr v-if="message.Cc && message.Cc.length" class="small">
							<th>Cc</th>
							<td class="privacy">
								<span v-for="(t, i) in message.Cc">
									<template v-if="i > 0">,</template>
									<span class="text-spaces">{{ t.Name }}</span>
									&lt;<a :href="searchURI(t.Address)" class="text-body">
										{{ t.Address }}
									</a>&gt;
								</span>
							</td>
						</tr>
						<tr v-if="message.Bcc && message.Bcc.length" class="small">
							<th>Bcc</th>
							<td class="privacy">
								<span v-for="(t, i) in message.Bcc">
									<template v-if="i > 0">,</template>
									<span class="text-spaces">{{ t.Name }}</span>
									&lt;<a :href="searchURI(t.Address)" class="text-body">
										{{ t.Address }}
									</a>&gt;
								</span>
							</td>
						</tr>
						<tr v-if="message.ReplyTo && message.ReplyTo.length" class="small">
							<th class="text-nowrap">Reply-To</th>
							<td class="privacy text-body-secondary text-break">
								<span v-for="(t, i) in message.ReplyTo">
									<template v-if="i > 0">,</template>
									<span class="text-spaces">{{ t.Name }}</span>
									&lt;<a :href="searchURI(t.Address)" class="text-body-secondary">
										{{ t.Address }}
									</a>&gt;
								</span>
							</td>
						</tr>
						<tr v-if="message.ReturnPath && message.From && message.ReturnPath != message.From.Address"
							class="small">
							<th class="text-nowrap">Return-Path</th>
							<td class="privacy text-body-secondary text-break">
								&lt;<a :href="searchURI(message.ReturnPath)" class="text-body-secondary">
									{{ message.ReturnPath }}
								</a>&gt;
							</td>
						</tr>
						<tr>
							<th class="small">Subject</th>
							<td>
								<strong v-if="message.Subject != ''" class="text-spaces">{{ message.Subject }}</strong>
								<small class="text-body-secondary" v-else>[ no subject ]</small>
							</td>
						</tr>
						<tr class="small">
							<th class="small">Date</th>
							<td>
								{{ messageDate(message.Date) }}
								<small class="ms-2">({{ getFileSize(message.Size) }})</small>
							</td>
						</tr>
						<tr class="small">
							<th>Tags</th>
							<td>
								<select class="form-select small tag-selector" v-model="messageTags" multiple
									data-full-width="false" data-suggestions-threshold="1" data-allow-new="true"
									data-clear-end="true" data-allow-clear="true" data-placeholder="Add tags..."
									data-badge-style="secondary" data-regex="^([a-zA-Z0-9\-\ \_\.]){1,}$"
									data-separator="|,|">
									<option value="">Type a tag...</option>
									<!-- you need at least one option with the placeholder -->
									<option v-for="t in availableTags" :value="t">{{ t }}</option>
								</select>
								<div class="invalid-feedback">Invalid tag name</div>
							</td>
						</tr>

						<tr v-if="message.ListUnsubscribe.Header != ''" class="small"
							:class="showUnsubscribe ? '' : 'd-none'">
							<th>Unsubscribe</th>
							<td>
								<span v-if="message.ListUnsubscribe.Links.length" class="text-secondary small me-2">
									<template v-for="(u, i) in message.ListUnsubscribe.Links">
										<template v-if="i > 0">, </template>
										&lt;{{ u }}&gt;
									</template>
								</span>
								<i class="bi bi-info-circle text-success me-2 link"
									v-if="message.ListUnsubscribe.HeaderPost != ''" data-bs-toggle="tooltip"
									data-bs-placement="top" data-bs-custom-class="custom-tooltip"
									:data-bs-title="'List-Unsubscribe-Post: ' + message.ListUnsubscribe.HeaderPost">
								</i>
								<i class="bi bi-exclamation-circle text-danger link"
									v-if="message.ListUnsubscribe.Errors != ''" data-bs-toggle="tooltip"
									data-bs-placement="top" data-bs-custom-class="custom-tooltip"
									:data-bs-title="message.ListUnsubscribe.Errors">
								</i>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
			<div class="col-md-auto d-none d-md-block text-end mt-md-3"
				v-if="message.Attachments && message.Attachments.length || message.Inline && message.Inline.length">
				<div class="mt-2 mt-md-0">
					<template v-if="message.Attachments.length">
						<span class="badge rounded-pill text-bg-secondary p-2 mb-2" title="Attachments in this message">
							Attachment<span v-if="message.Attachments.length > 1">s</span>
							({{ message.Attachments.length }})
						</span>
						<br>
					</template>
					<span class="badge rounded-pill text-bg-secondary p-2" v-if="message.Inline.length"
						title="Inline images in this message">
						Inline image<span v-if="message.Inline.length > 1">s</span>
						({{ message.Inline.length }})
					</span>
				</div>
			</div>
		</div>

		<nav class="nav nav-tabs my-3 d-print-none" id="nav-tab" role="tablist">
			<template v-if="message.HTML">
				<div class="btn-group">
					<button class="nav-link" id="nav-html-tab" data-bs-toggle="tab" data-bs-target="#nav-html"
						type="button" role="tab" aria-controls="nav-html" aria-selected="true" ref="navhtml"
						v-on:click="resizeIFrames()">
						HTML
					</button>
					<button type="button" class="nav-link dropdown-toggle dropdown-toggle-split d-sm-none"
						data-bs-toggle="dropdown" aria-expanded="false" data-bs-reference="parent">
						<span class="visually-hidden">Toggle Dropdown</span>
					</button>
					<div class="dropdown-menu">
						<button class="dropdown-item" data-bs-toggle="tab" data-bs-target="#nav-html-source"
							type="button" role="tab" aria-controls="nav-html-source" aria-selected="false">
							HTML Source
						</button>
					</div>
				</div>
				<button class="nav-link d-none d-sm-inline" id="nav-html-source-tab" data-bs-toggle="tab"
					data-bs-target="#nav-html-source" type="button" role="tab" aria-controls="nav-html-source"
					aria-selected="false">
					HTML <span class="d-sm-none">Src</span><span class="d-none d-sm-inline">Source</span>
				</button>
			</template>

			<button class="nav-link" id="nav-plain-text-tab" data-bs-toggle="tab" data-bs-target="#nav-plain-text"
				type="button" role="tab" aria-controls="nav-plain-text" aria-selected="false"
				:class="message.HTML == '' ? 'show' : ''">
				Text
			</button>
			<button class="nav-link" id="nav-headers-tab" data-bs-toggle="tab" data-bs-target="#nav-headers"
				type="button" role="tab" aria-controls="nav-headers" aria-selected="false">
				<span class="d-sm-none">Hdrs</span><span class="d-none d-sm-inline">Headers</span>
			</button>
			<button class="nav-link" id="nav-raw-tab" data-bs-toggle="tab" data-bs-target="#nav-raw" type="button"
				role="tab" aria-controls="nav-raw" aria-selected="false">
				Raw
			</button>
			<div class="dropdown d-xl-none" v-show="hasAnyChecksEnabled">
				<button class="nav-link dropdown-toggle" type="button" data-bs-toggle="dropdown" aria-expanded="false">
					Checks
				</button>
				<ul class="dropdown-menu checks">
					<li v-if="mailbox.showHTMLCheck && message.HTML != ''">
						<button class="dropdown-item" id="nav-html-check-tab" data-bs-toggle="tab"
							data-bs-target="#nav-html-check" type="button" role="tab" aria-controls="nav-html"
							aria-selected="false">
							HTML Check
							<span class="badge rounded-pill p-1 float-end" :class="htmlScoreColor"
								v-if="htmlScore !== false">
								<small>{{ Math.floor(htmlScore) }}%</small>
							</span>
						</button>
					</li>
					<li v-if="mailbox.showLinkCheck">
						<button class="dropdown-item" id="nav-link-check-tab" data-bs-toggle="tab"
							data-bs-target="#nav-link-check" type="button" role="tab" aria-controls="nav-link-check"
							aria-selected="false">
							Link Check
							<span class="badge rounded-pill bg-success float-end" v-if="linkCheckErrors === 0">
								<small>0</small>
							</span>
							<span class="badge rounded-pill bg-danger float-end" v-else-if="linkCheckErrors > 0">
								<small>{{ formatNumber(linkCheckErrors) }}</small>
							</span>
						</button>
					</li>
					<li v-if="mailbox.showSpamCheck && mailbox.uiConfig.SpamAssassin">
						<button class="dropdown-item" id="nav-spam-check-tab" data-bs-toggle="tab"
							data-bs-target="#nav-spam-check" type="button" role="tab" aria-controls="nav-html"
							aria-selected="false">
							Spam Analysis
							<span class="badge rounded-pill float-end" :class="spamScoreColor"
								v-if="spamScore !== false">
								<small>{{ spamScore }}</small>
							</span>
						</button>
					</li>
				</ul>
			</div>
			<button class="d-none d-xl-inline-block nav-link position-relative" id="nav-html-check-tab"
				data-bs-toggle="tab" data-bs-target="#nav-html-check" type="button" role="tab" aria-controls="nav-html"
				aria-selected="false" v-if="mailbox.showHTMLCheck && message.HTML != ''">
				HTML Check
				<span class="badge rounded-pill p-1" :class="htmlScoreColor" v-if="htmlScore !== false">
					<small>{{ Math.floor(htmlScore) }}%</small>
				</span>
			</button>
			<button class="d-none d-xl-inline-block nav-link" id="nav-link-check-tab" data-bs-toggle="tab"
				data-bs-target="#nav-link-check" type="button" role="tab" aria-controls="nav-link-check"
				aria-selected="false" v-if="mailbox.showLinkCheck">
				Link Check
				<i class="bi bi-check-all text-success" v-if="linkCheckErrors === 0"></i>
				<span class="badge rounded-pill bg-danger" v-else-if="linkCheckErrors > 0">
					<small>{{ formatNumber(linkCheckErrors) }}</small>
				</span>
			</button>
			<button class="d-none d-xl-inline-block nav-link position-relative" id="nav-spam-check-tab"
				data-bs-toggle="tab" data-bs-target="#nav-spam-check" type="button" role="tab" aria-controls="nav-html"
				aria-selected="false" v-if="mailbox.showSpamCheck && mailbox.uiConfig.SpamAssassin">
				Spam Analysis
				<span class="badge rounded-pill" :class="spamScoreColor" v-if="spamScore !== false">
					<small>{{ spamScore }}</small>
				</span>
			</button>

			<div class="d-none d-lg-block ms-auto me-3" v-if="showMobileButtons">
				<template v-for="_, key in responsiveSizes">
					<button class="btn" :disabled="scaleHTMLPreview == key" :title="'Switch to ' + key + ' view'"
						v-on:click="scaleHTMLPreview = key">
						<i class="bi" :class="'bi-' + key"></i>
					</button>
				</template>
			</div>
		</nav>

		<div class="tab-content mb-5" id="nav-tabContent">
			<div v-if="message.HTML != ''" class="tab-pane fade show" id="nav-html" role="tabpanel"
				aria-labelledby="nav-html-tab" tabindex="0">
				<div id="responsive-view" :class="scaleHTMLPreview" :style="responsiveSizes[scaleHTMLPreview]">
					<iframe target-blank="" class="tab-pane d-block" id="preview-html" :srcdoc="sanitizedHTML"
						v-on:load="resizeIframe" frameborder="0" style="width: 100%; height: 100%; background: #fff;">
					</iframe>
				</div>
				<Attachments v-if="allAttachments(message).length" :message="message"
					:attachments="allAttachments(message)">
				</Attachments>
			</div>
			<div class="tab-pane fade" id="nav-html-source" role="tabpanel" aria-labelledby="nav-html-source-tab"
				tabindex="0" v-if="message.HTML">
				<pre><code class="language-html">{{ message.HTML }}</code></pre>
			</div>
			<div class="tab-pane fade" id="nav-plain-text" role="tabpanel" aria-labelledby="nav-plain-text-tab"
				tabindex="0" :class="message.HTML == '' ? 'show' : ''">
				<div class="text-view" v-html="textToHTML(message.Text)"></div>
				<Attachments v-if="allAttachments(message).length" :message="message"
					:attachments="allAttachments(message)">
				</Attachments>
			</div>
			<div class="tab-pane fade" id="nav-headers" role="tabpanel" aria-labelledby="nav-headers-tab" tabindex="0">
				<Headers v-if="loadHeaders" :message="message"></Headers>
			</div>
			<div class="tab-pane fade" id="nav-raw" role="tabpanel" aria-labelledby="nav-raw-tab" tabindex="0">
				<iframe v-if="srcURI" :src="srcURI" v-on:load="initRawIframe" frameborder="0"
					style="width: 100%; height: 300px"></iframe>
			</div>
			<div class="tab-pane fade" id="nav-html-check" role="tabpanel" aria-labelledby="nav-html-check-tab"
				tabindex="0">
				<HTMLCheck v-if="mailbox.showHTMLCheck && message.HTML != ''" :message="message"
					@setHtmlScore="(n) => htmlScore = n" @set-badge-style="(v) => htmlScoreColor = v" />
			</div>
			<div class="tab-pane fade" id="nav-spam-check" role="tabpanel" aria-labelledby="nav-spam-check-tab"
				tabindex="0" v-if="mailbox.showSpamCheck && mailbox.uiConfig.SpamAssassin">
				<SpamAssassin :message="message" @setSpamScore="(n) => spamScore = n"
					@set-badge-style="(v) => spamScoreColor = v" />
			</div>
			<div class="tab-pane fade" id="nav-link-check" role="tabpanel" aria-labelledby="nav-html-check-tab"
				tabindex="0" v-if="mailbox.showLinkCheck">
				<LinkCheck :message="message" @setLinkErrors="(n) => linkCheckErrors = n" />
			</div>
		</div>
	</div>
</template>
