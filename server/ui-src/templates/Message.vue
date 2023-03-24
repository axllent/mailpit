
<script>
import commonMixins from '../mixins.js';
import Prism from "prismjs";
import Tags from "bootstrap5-tags";
import Attachments from './Attachments.vue';

export default {
	props: {
		message: Object,
		existingTags: Array
	},

	components: {
		Attachments
	},

	mixins: [commonMixins],

	data() {
		return {
			srcURI: false,
			iframes: [], // for resizing
			showTags: false, // to force rerendering of component
			messageTags: [],
			allTags: [],
		}
	},

	watch: {
		message: {
			handler() {
				let self = this;
				self.showTags = false;
				self.messageTags = self.message.Tags;
				self.allTags = self.existingTags;
				// delay to select first tab and add HTML highlighting (prev/next)
				self.$nextTick(function () {
					self.renderUI();
					self.showTags = true;
					self.$nextTick(function () {
						Tags.init("select[multiple]");
					});
				});
			},
			// force eager callback execution
			immediate: true
		},
		messageTags() {
			// save changed to tags
			if (this.showTags) {
				this.saveTags();
			}
		}
	},

	mounted() {
		let self = this;
		self.showTags = false;
		self.allTags = self.existingTags;
		window.addEventListener("resize", self.resizeIframes);
		self.renderUI();
		var tabEl = document.getElementById('nav-raw-tab');
		tabEl.addEventListener('shown.bs.tab', function (event) {
			self.srcURI = 'api/v1/message/' + self.message.ID + '/raw';
		});

		self.showTags = true;
		self.$nextTick(function () {
			Tags.init("select[multiple]");
		});
	},

	unmounted: function () {
		window.removeEventListener("resize", this.resizeIframes);
	},

	methods: {
		renderUI: function () {
			let self = this;
			// click the first non-disabled tab
			document.querySelector('#nav-tab button:not([disabled])').click();
			document.activeElement.blur(); // blur focus
			document.getElementById('message-view').scrollTop = 0;

			// delay 0.2s until vue has rendered the iframe content
			window.setTimeout(function () {
				let p = document.getElementById('preview-html');
				if (p) {
					// make links open in new window
					let anchorEls = p.contentWindow.document.body.querySelectorAll('a');
					for (var i = 0; i < anchorEls.length; i++) {
						let anchorEl = anchorEls[i];
						let href = anchorEl.getAttribute('href');

						if (href && href.match(/^http/)) {
							anchorEl.setAttribute('target', '_blank');
						}
					}
					self.resizeIframes();
				}
			}, 200);

			// html highlighting
			window.Prism = window.Prism || {};
			window.Prism.manual = true;
			Prism.highlightAll();
		},

		resizeIframe: function (el) {
			let i = el.target;
			i.style.height = i.contentWindow.document.body.scrollHeight + 50 + 'px';
		},

		resizeIframes: function () {
			let h = document.getElementById('preview-html');
			if (h) {
				h.style.height = h.contentWindow.document.body.scrollHeight + 50 + 'px';
			}

			let s = document.getElementById('message-src');
			if (s) {
				s.style.height = s.contentWindow.document.body.scrollHeight + 50 + 'px';
			}
		},

		saveTags: function () {
			let self = this;

			var data = {
				ids: [this.message.ID],
				tags: this.messageTags
			}

			self.put('api/v1/tags', data, function (response) {
				self.scrollInPlace = true;
				self.$emit('loadMessages');
			});
		}
	}
}
</script>

<template>
	<div v-if="message" id="message-view" class="mh-100" style="overflow-y: scroll;">
		<div class="row w-100">
			<div class="col-md">
				<table class="messageHeaders">
					<tbody>
						<tr class="small">
							<th>From</th>
							<td class="privacy">
								<span v-if="message.From">
									<span v-if="message.From.Name">{{ message.From.Name + " " }}</span>
									<span v-if="message.From.Address">&lt;{{ message.From.Address }}&gt;</span>
								</span>
								<span v-else>
									[ Unknown ]
								</span>
							</td>
						</tr>
						<tr class="small">
							<th>To</th>
							<td class="privacy">
								<span v-if="message.To && message.To.length" v-for="(t, i) in message.To">
									<template v-if="i > 0">, </template>
									<span class="text-nowrap">{{ t.Name + " <" + t.Address + ">" }}</span>
									</span>
									<span v-else>Undisclosed recipients</span>
							</td>
						</tr>
						<tr v-if="message.Cc && message.Cc.length" class="small">
							<th>Cc</th>
							<td class="privacy">
								<span v-for="(t, i) in message.Cc">
									<template v-if="i > 0">,</template>
									{{ t.Name + " <" + t.Address + ">" }} </span>
							</td>
						</tr>
						<tr v-if="message.Bcc && message.Bcc.length" class="small">
							<th>Bcc</th>
							<td class="privacy">
								<span v-for="(t, i) in message.Bcc">
									<template v-if="i > 0">,</template>
									{{ t.Name + " <" + t.Address + ">" }} </span>
							</td>
						</tr>
						<tr>
							<th class="small">Subject</th>
							<td><strong>{{ message.Subject }}</strong></td>
						</tr>
						<tr class="d-md-none small">
							<th class="small">Date</th>
							<td>{{ messageDate(message.Date) }}</td>
						</tr>

						<tr class="small" v-if="showTags">
							<th>Tags</th>
							<td>
								<select class="form-select small tag-selector" v-model="messageTags" multiple
									data-allow-new="true" data-clear-end="true" data-allow-clear="true"
									data-placeholder="Add tags..." data-badge-style="secondary"
									data-regex="^([a-zA-Z0-9\-\ \_]){3,}$" data-separator="|,|">
									<option value="">Type a tag...</option>
									<!-- you need at least one option with the placeholder -->
									<option v-for="t in allTags" :value="t">{{ t }}</option>
								</select>
								<div class="invalid-feedback">Please select a valid tag.</div>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
			<div class="col-md-auto text-md-end mt-md-3">
				<!-- <p class="text-muted small d-none d-md-block mb-2"><small>{{ messageDate(message.Date) }}</small></p>
																																																																												<p class="text-muted small d-none d-md-block"><small>Size: {{ getFileSize(message.Size) }}</small></p> -->
				<div class="dropdown mt-2 mt-md-0" v-if="allAttachments(message)">
					<button class="btn btn-outline-secondary dropdown-toggle" type="button" data-bs-toggle="dropdown"
						aria-expanded="false">
						Attachment<span v-if="allAttachments(message).length > 1">s</span>
						({{ allAttachments(message).length }})
					</button>
					<ul class="dropdown-menu">
						<li v-for="part in allAttachments(message)">
							<a :href="'api/v1/message/' + message.ID + '/part/' + part.PartID" type="button"
								class="dropdown-item" target="_blank">
								<i class="bi" :class="attachmentIcon(part)"></i>
								{{ part.FileName != '' ? part.FileName : '[ unknown ]' }}
								<small class="text-muted ms-2">{{ getFileSize(part.Size) }}</small>
							</a>
						</li>
					</ul>
				</div>
			</div>
		</div>

		<nav>
			<div class="nav nav-tabs my-3" id="nav-tab" role="tablist">
				<button class="nav-link" id="nav-html-tab" data-bs-toggle="tab" data-bs-target="#nav-html" type="button"
					role="tab" aria-controls="nav-html" aria-selected="true" v-if="message.HTML">HTML</button>
				<button class="nav-link" id="nav-html-source-tab" data-bs-toggle="tab" data-bs-target="#nav-html-source"
					type="button" role="tab" aria-controls="nav-html-source" aria-selected="false" v-if="message.HTML">HTML
					Source</button>
				<button class="nav-link" id="nav-plain-text-tab" data-bs-toggle="tab" data-bs-target="#nav-plain-text"
					type="button" role="tab" aria-controls="nav-plain-text" aria-selected="false"
					:class="message.HTML == '' ? 'show' : ''">Text</button>
				<button class="nav-link" id="nav-raw-tab" data-bs-toggle="tab" data-bs-target="#nav-raw" type="button"
					role="tab" aria-controls="nav-raw" aria-selected="false">Raw</button>
			</div>
		</nav>
		<div class="tab-content mb-5" id="nav-tabContent">
			<div v-if="message.HTML != ''" class="tab-pane fade show" id="nav-html" role="tabpanel"
				aria-labelledby="nav-html-tab" tabindex="0">
				<iframe target-blank="" class="tab-pane" id="preview-html" :srcdoc="message.HTML" v-on:load="resizeIframe"
					seamless frameborder="0" style="width: 100%; height: 100%;">
				</iframe>
				<Attachments v-if="allAttachments(message).length" :message="message"
					:attachments="allAttachments(message)"></Attachments>
			</div>
			<div class="tab-pane fade" id="nav-html-source" role="tabpanel" aria-labelledby="nav-html-source-tab"
				tabindex="0" v-if="message.HTML">
				<pre><code class="language-html">{{ message.HTML }}</code></pre>
			</div>
			<div class="tab-pane fade" id="nav-plain-text" role="tabpanel" aria-labelledby="nav-plain-text-tab" tabindex="0"
				:class="message.HTML == '' ? 'show' : ''">
				<div class="text-view">{{ message.Text }}</div>
				<Attachments v-if="allAttachments(message).length" :message="message"
					:attachments="allAttachments(message)"></Attachments>
			</div>
			<div class="tab-pane fade" id="nav-raw" role="tabpanel" aria-labelledby="nav-raw-tab" tabindex="0">
				<iframe v-if="srcURI" :src="srcURI" v-on:load="resizeIframe" seamless frameborder="0"
					style="width: 100%; height: 300px;" id="message-src"></iframe>
			</div>
		</div>
	</div>
</template>
