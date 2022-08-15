
<script>
import commonMixins from '../mixins.js';
import moment from 'moment'

export default {
	props: {
		message: Object,
		mailbox: Object,
	},
	mixins: [commonMixins],
	data() {
		return {
			srcURI: false,
			iframes: [], // for resizing
		}
	},

	mounted() {
		var self = this;

		window.addEventListener("resize", self.resizeIframes);

		// click the first non-disabled tab
		document.querySelector('#nav-tab button:not([disabled])').click();
		document.activeElement.blur(); // blur focus

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
			}
		}, 200);

		var tabEl = document.getElementById('nav-source-tab');
		tabEl.addEventListener('shown.bs.tab', function (event) {
			self.srcURI = 'api/' + self.mailbox + '/' + self.message.ID + '/source';
		});
	},
	
	unmounted: function() {
		window.removeEventListener("resize", this.resizeIframes);
	},

	methods: {
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
	<div v-if="message" class="mh-100" style="overflow-y: scroll;">
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
						<span v-for="(t, i) in message.To">
							<template v-if="i > 0">,</template>
							{{ t.Name + " <" + t.Address +">" }}
						</span>
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

		<nav>
			<div class="nav nav-tabs my-3" id="nav-tab" role="tablist">
				<button class="nav-link" id="nav-html-tab" data-bs-toggle="tab"
					data-bs-target="#nav-html" type="button" role="tab" aria-controls="nav-html"
					aria-selected="true" :disabled="message.HTML == ''" :class="message.HTML == '' ? 'disabled':''">HTML</button>
				<button class="nav-link" id="nav-plain-text-tab" data-bs-toggle="tab"
					data-bs-target="#nav-plain-text" type="button" role="tab" aria-controls="nav-plain-text"
					aria-selected="false" :class="message.HTML == '' ? 'show':''">Plain<span class="d-none d-md-inline"> text</span></button>
				<button class="nav-link" id="nav-source-tab" data-bs-toggle="tab"
					data-bs-target="#nav-source" type="button" role="tab" aria-controls="nav-source"
					aria-selected="false">Source</button>
				<button class="nav-link" id="nav-mime-tab" data-bs-toggle="tab" data-bs-target="#nav-mime"
					type="button" role="tab" aria-controls="nav-mime" aria-selected="false"
					:disabled="!allAttachments(message)" :class="!allAttachments(message) ? 'disabled':''"
					>Attachments <span v-if="allAttachments(message)">({{allAttachments(message).length}})</span></button>
				<div class="d-none d-lg-block ms-auto small mt-3 me-2 text-muted">
					<small>{{ messageDate(message.Date) }}</small>
				</div>
			</div>
		</nav>
		<div class="tab-content mb-5" id="nav-tabContent">
			<div v-if="message.HTML != ''" class="tab-pane fade show" id="nav-html" role="tabpanel"
				aria-labelledby="nav-html-tab" tabindex="0">
				<iframe target-blank="" class="tab-pane" id="preview-html" :srcdoc="message.HTML" v-on:load="resizeIframe"
					seamless frameborder="0" style="width: 100%; height: 100%;">
				</iframe>
			</div>
			<div class="tab-pane fade" id="nav-plain-text" role="tabpanel"
				aria-labelledby="nav-plain-text-tab" tabindex="0" :class="message.HTML == '' ? 'show':''">
				{{ message.Text }}
			</div>
			<div class="tab-pane fade" id="nav-source" role="tabpanel" aria-labelledby="nav-source-tab"
				tabindex="0">
				<iframe v-if="srcURI" :src="srcURI" v-on:load="resizeIframe"
					seamless frameborder="0" style="width: 100%; height: 300px;" id="message-src"></iframe>
			</div>
			<div class="tab-pane fade" id="nav-mime" role="tabpanel" aria-labelledby="nav-mime-tab"
				tabindex="0">
				<div v-if="allAttachments(message)" v-for="part in allAttachments(message)" class="mime-part mb-2">
					<a :href="'api/'+mailbox+'/'+message.ID+'/part/'+part.PartID" type="button"
						class="btn btn-outline-secondary btn-sm me-2" target="_blank">
						<i class="bi bi-file-arrow-down-fill"></i>
						{{ part.FileName != '' ? part.FileName : '[ unknown ]' }}
					</a>
					<small class="text-muted">{{ getFileSize(part.Size) }}</small>
				</div>
			</div>
		</div>
	</div>
</template>
