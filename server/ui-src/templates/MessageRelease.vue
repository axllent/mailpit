
<script>
import Tags from "bootstrap5-tags"
import commonMixins from '../mixins.js'

export default {
	props: {
		message: Object,
		uiConfig: Object,
		releaseAddresses: Array
	},

	data() {
		return {
			addresses: []
		}
	},

	mixins: [commonMixins],

	mounted() {
		this.addresses = JSON.parse(JSON.stringify(this.releaseAddresses))
		this.$nextTick(function () {
			Tags.init("select[multiple]")
		})
	},

	methods: {
		releaseMessage: function () {
			let self = this
			// set timeout to allow for user clicking send before the tag filter has applied the tag
			window.setTimeout(function () {
				if (!self.addresses.length) {
					return false
				}

				let data = {
					to: self.addresses
				}

				self.post('api/v1/message/' + self.message.ID + '/release', data, function (response) {
					self.modal("ReleaseModal").hide()
				})
			}, 100)
		}
	}
}
</script>

<template>
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
							:value="message.From.Address">
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
							data-clear-end="true" data-allow-clear="true" data-placeholder="Enter email addresses..."
							data-add-on-blur="true" data-badge-style="primary"
							data-regex='^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|.(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$'
							data-separator="|,|">
							<option value="">Enter email addresses...</option>
							<!-- you need at least one option with the placeholder -->
							<option v-for="t in releaseAddresses" :value="t">{{ t }}</option>
						</select>
						<div class="invalid-feedback">Invalid email address</div>
					</div>
				</div>
				<div class="form-text text-center" v-if="uiConfig.MessageRelay.RecipientAllowlist != ''">
					Note: A recipient allowlist has been configured. Any mail address not matching it will be rejected.
					<br class="d-none d-md-inline">
					Configured allowlist: <b>{{ uiConfig.MessageRelay.RecipientAllowlist }}</b>
				</div>
				<div class="form-text text-center">
					Note: For testing purposes, a unique Message-Id will be generated on send.
					<br class="d-none d-md-inline">
					SMTP delivery failures will bounce back to
					<b v-if="uiConfig.MessageRelay.ReturnPath != ''">{{ uiConfig.MessageRelay.ReturnPath }}</b>
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

	<div id="loading" v-if="loading">
		<div class="d-flex justify-content-center align-items-center h-100">
			<div class="spinner-border text-secondary" role="status">
				<span class="visually-hidden">Loading...</span>
			</div>
		</div>
	</div>
</template>
