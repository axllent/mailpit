
<script>
import commonMixins from '../mixins.js';
import moment from 'moment';
import Prism from "prismjs";

export default {
	props: {
		message: Object
	},

	mixins: [commonMixins],

	data() {
		return {
			srcURI: false,
			iframes: [], // for resizing
		}
	},

	watch: {
		message: {
			handler(newQuestion) {
				let self = this;
				// delay 100ms to select first tab and add HTML highlighting (prev/next)
				window.setTimeout(function() {
					self.renderUI();
				}, 100)
			},
			// force eager callback execution
			immediate: true
		}

	},

	mounted() {
		let self = this;
		window.addEventListener("resize", self.resizeIframes);
		self.renderUI();
		var tabEl = document.getElementById('nav-raw-tab');
		tabEl.addEventListener('shown.bs.tab', function (event) {
			self.srcURI = 'api/' + self.message.ID + '/raw';
		});
	},
	
	unmounted: function() {
		window.removeEventListener("resize", this.resizeIframes);
	},

	methods: {
		renderUI: function() {
			let self = this;
			// click the first non-disabled tab
			document.querySelector('#nav-tab button:not([disabled])').click();
			document.activeElement.blur(); // blur focus
			document.getElementById('message-view').scrollTop = 0;

			window.setTimeout(function(){
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
		
		resizeIframe: function(el) {
			let i = el.target;
			i.style.height = i.contentWindow.document.body.scrollHeight + 50 + 'px';
		},

		resizeIframes: function() {
			let h = document.getElementById('preview-html');
			if (h) {
				h.style.height = h.contentWindow.document.body.scrollHeight + 50 + 'px';
			}

			let s = document.getElementById('message-src');
			if (s) {
				s.style.height = s.contentWindow.document.body.scrollHeight + 50 + 'px';
			}
		},

		allAttachments: function(message){
			let a = [];
			for (let i in message.Attachments) {
				a.push(message.Attachments[i]);
			}
			for (let i in message.OtherParts) {
				a.push(message.OtherParts[i]);
			}
			for (let i in message.Inline) {
				a.push(message.Inline[i]);
			}
			
			return a.length ? a : false;
		},

		messageDate: function(d) {
			return moment(d).format('ddd, D MMM YYYY, h:mm a');
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
								<span v-if="message.To" v-for="(t, i) in message.To">
									<template v-if="i > 0">,</template>
									{{ t.Name + " <" + t.Address +">" }}
								</span>
								<span v-else>Undisclosed recipients</span>
							</td>
						</tr>
						<tr v-if="message.Cc" class="small">
							<th>CC</th>
							<td class="privacy">
								<span v-for="(t, i) in message.Cc">
									<template v-if="i > 0">,</template>
									{{ t.Name + " <" + t.Address +">" }}
								</span>
							</td>
						</tr>
						<tr v-if="message.Bcc" class="small">
							<th>CC</th>
							<td class="privacy">
								<span v-for="(t, i) in message.Bcc">
									<template v-if="i > 0">,</template>
									{{ t.Name + " <" + t.Address +">" }}
								</span>
							</td>
						</tr>
						<tr>
							<th class="small">Subject</th>
							<td><strong>{{ message.Subject }}</strong></td>
						</tr>
					</tbody>
				</table>
			</div>
			<div class="col-md-auto text-md-end mt-md-3">
				<p class="text-muted small"><small>{{ messageDate(message.Date) }}</small></p>
				<div class="dropdown" v-if="allAttachments(message)">
					<button class="btn btn-outline-secondary dropdown-toggle" type="button" data-bs-toggle="dropdown" aria-expanded="false">
						Attachment<span v-if="allAttachments(message).length > 1">s</span>
						({{ allAttachments(message).length }})
					</button>
					<ul class="dropdown-menu">
						<li v-for="part in allAttachments(message)">
							<a :href="'api/'+message.ID+'/part/'+part.PartID" type="button"
								class="dropdown-item" target="_blank">
								<i class="bi bi-file-arrow-down-fill"></i>
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
				<button class="nav-link" id="nav-html-tab" data-bs-toggle="tab"
					data-bs-target="#nav-html" type="button" role="tab" aria-controls="nav-html"
					aria-selected="true" v-if="message.HTML">HTML</button>
				<button class="nav-link" id="nav-html-source-tab" data-bs-toggle="tab"
					data-bs-target="#nav-html-source" type="button" role="tab" aria-controls="nav-html-source"
					aria-selected="false" v-if="message.HTMLSource">HTML Source</button>
				<button class="nav-link" id="nav-plain-text-tab" data-bs-toggle="tab"
					data-bs-target="#nav-plain-text" type="button" role="tab" aria-controls="nav-plain-text"
					aria-selected="false" :class="message.HTML == '' ? 'show':''">Text</button>
				<button class="nav-link" id="nav-raw-tab" data-bs-toggle="tab"
					data-bs-target="#nav-raw" type="button" role="tab" aria-controls="nav-raw"
					aria-selected="false">Raw</button>
			</div>
		</nav>
		<div class="tab-content mb-5" id="nav-tabContent">
			<div v-if="message.HTML != ''" class="tab-pane fade show" id="nav-html" role="tabpanel"
				aria-labelledby="nav-html-tab" tabindex="0">
				<iframe target-blank="" class="tab-pane" id="preview-html" :srcdoc="message.HTML" v-on:load="resizeIframe"
					seamless frameborder="0" style="width: 100%; height: 100%;">
				</iframe>
			</div>
			<div class="tab-pane fade" id="nav-html-source" role="tabpanel"
				aria-labelledby="nav-html-source-tab" tabindex="0" v-if="message.HTMLSource">
				<pre><code class="language-html">{{ message.HTMLSource }}</code></pre>
			</div>
			<div class="tab-pane fade" id="nav-plain-text" role="tabpanel"
				aria-labelledby="nav-plain-text-tab" tabindex="0" :class="message.HTML == '' ? 'show':''">
				{{ message.Text }}
			</div>
			<div class="tab-pane fade" id="nav-raw" role="tabpanel" aria-labelledby="nav-raw-tab"
				tabindex="0">
				<iframe v-if="srcURI" :src="srcURI" v-on:load="resizeIframe"
					seamless frameborder="0" style="width: 100%; height: 300px;" id="message-src"></iframe>
			</div>
		</div>
	</div>
</template>
