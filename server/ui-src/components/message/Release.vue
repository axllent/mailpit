<script>
import AjaxLoader from '../AjaxLoader.vue'
import Tags from "bootstrap5-tags"
import commonMixins from '../../mixins/CommonMixins'
import { mailbox } from '../../stores/mailbox'

export default {
	props: {
		message: Object,
	},

	components: {
		AjaxLoader,
	},

	emits: ['delete'],

	data() {
		return {
			addresses: [],
			deleteAfterRelease: false,
			mailbox,
			allAddresses: [],
		}
	},

	mixins: [commonMixins],

	mounted() {
		let a = []
		for (let i in this.message.To) {
			a.push(this.message.To[i].Address)
		}
		for (let i in this.message.Cc) {
			a.push(this.message.Cc[i].Address)
		}
		for (let i in this.message.Bcc) {
			a.push(this.message.Bcc[i].Address)
		}

		// include only unique email addresses, regardless of casing
		this.allAddresses = JSON.parse(JSON.stringify([...new Map(a.map(ad => [ad.toLowerCase(), ad])).values()]))

		this.addresses = this.allAddresses
	},

	methods: {
		// triggered manually after modal is shown
		initTags: function () {
			Tags.init("select[multiple]")
		},

		releaseMessage: function () {
			let self = this
			// set timeout to allow for user clicking send before the tag filter has applied the tag
			window.setTimeout(function () {
				if (!self.addresses.length) {
					return false
				}

				let data = {
					To: self.addresses
				}

				self.post(self.resolve('/api/v1/message/' + self.message.ID + '/release'), data, function (response) {
					self.modal("ReleaseModal").hide()
					if (self.deleteAfterRelease) {
						self.$emit('delete')
					}
				})
			}, 100)
		}
	}
}
</script>

<template>
	<div class="modal fade" id="ReleaseModal" tabindex="-1" aria-labelledby="AppInfoModalLabel" aria-hidden="true">
		<div class="modal-dialog modal-lg" v-if="message">
			<div class="modal-content">
				<div class="modal-header">
					<h1 class="modal-title fs-5" id="AppInfoModalLabel">Release email</h1>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div class="modal-body">
					<h6>Send this message to one or more addresses specified below.</h6>
					<div class="row">
						<label class="col-sm-2 col-form-label text-body-secondary">From</label>
						<div class="col-sm-10">
							<input type="text" aria-label="From address" readonly class="form-control-plaintext"
								:value="message.From ? message.From.Address : ''">
						</div>
					</div>
					<div class="row">
						<label class=" col-sm-2 col-form-label text-body-secondary">Subject</label>
						<div class="col-sm-10">
							<input type="text" aria-label="Subject" readonly class="form-control-plaintext"
								:value="message.Subject">
						</div>
					</div>
					<div class="row mb-3">
						<label class="col-sm-2 col-form-label text-body-secondary">Send to</label>
						<div class="col-sm-10">
							<select class="form-select tag-selector" v-model="addresses" multiple data-allow-new="true"
								data-clear-end="true" data-allow-clear="true"
								data-placeholder="Enter email addresses..." data-add-on-blur="true"
								data-badge-style="primary"
								data-regex='^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|.(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$'
								data-separator="|,|">
								<option value="">Enter email addresses...</option>
								<!-- you need at least one option with the placeholder -->
								<option v-for="t in allAddresses" :value="t">{{ t }}</option>
							</select>
							<div class="invalid-feedback">Invalid email address</div>
						</div>
					</div>
					<div class="row mb-3">
						<div class="col-sm-10 offset-sm-2">
							<div class="form-check">
								<input class="form-check-input" type="checkbox" v-model="deleteAfterRelease"
									id="DeleteAfterRelease">
								<label class="form-check-label" for="DeleteAfterRelease">
									Delete the message after release
								</label>
							</div>

						</div>
					</div>
					<div class="form-text text-center" v-if="mailbox.uiConfig.MessageRelay.AllowedRecipients != ''">
						Note: A recipient allowlist has been configured. Any mail address not matching it will be
						rejected.<br class="d-none d-md-inline">
						Allowed recipients: <b>{{ mailbox.uiConfig.MessageRelay.AllowedRecipients }}</b>
					</div>
					<div class="form-text text-center">
						Note: For testing purposes, a unique Message-Id will be generated on send.
						<br class="d-none d-md-inline">
						SMTP delivery failures will bounce back to
						<b v-if="mailbox.uiConfig.MessageRelay.ReturnPath != ''">
							{{ mailbox.uiConfig.MessageRelay.ReturnPath }}
						</b>
						<b v-else>{{ message.ReturnPath }}</b>.
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
					<button type="button" class="btn btn-primary" :disabled="!addresses.length"
						v-on:click="releaseMessage">Release</button>
				</div>
			</div>
		</div>
	</div>

	<AjaxLoader :loading="loading" />
</template>
