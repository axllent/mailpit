<script>
import CommonMixins from '../mixins/CommonMixins'
import Tags from 'bootstrap5-tags'
import timezones from 'timezones-list'
import { mailbox } from '../stores/mailbox'

export default {
	mixins: [CommonMixins],

	data() {
		return {
			mailbox,
			theme: localStorage.getItem('theme') ? localStorage.getItem('theme') : 'auto',
			timezones,
		}
	},

	watch: {
		theme: function (v) {
			if (v == 'auto') {
				localStorage.removeItem('theme')
			} else {
				localStorage.setItem('theme', v)
			}
			this.setTheme()
		}
	},

	mounted() {
		this.setTheme()
		this.$nextTick(function () {
			Tags.init('select.tz')
		})
	},

	methods: {
		setTheme: function () {
			if (
				this.theme === 'auto' &&
				window.matchMedia('(prefers-color-scheme: dark)').matches
			) {
				document.documentElement.setAttribute('data-bs-theme', 'dark')
			} else {
				document.documentElement.setAttribute('data-bs-theme', this.theme)
			}
		},
	}
}
</script>

<template>
	<div class="modal fade" id="SettingsModal" tabindex="-1" aria-labelledby="SettingsModalLabel" aria-hidden="true"
		data-bs-keyboard="false">
		<div class="modal-dialog modal-lg">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title" id="SettingsModalLabel">Mailpit UI settings</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div class="modal-body">
					<div class="mb-3">
						<label for="theme" class="form-label">Mailpit theme</label>
						<select class="form-select" v-model="theme" id="theme">
							<option value="auto">Auto (detect from browser)</option>
							<option value="light">Light theme</option>
							<option value="dark">Dark theme</option>
						</select>
					</div>
					<div class="mb-3">
						<label for="timezone" class="form-label">Timezone (for date searches)</label>
						<select class="form-select tz" v-model="mailbox.timeZone" id="timezone" data-allow-same="true">
							<option disabled hidden value="">Select a timezone...</option>
							<option v-for="t in timezones" :value="t.tzCode">{{ t.label }}</option>
						</select>
					</div>
					<div class="mb-3">
						<div class="form-check form-switch">
							<input class="form-check-input" type="checkbox" role="switch" id="tagColors"
								v-model="mailbox.showTagColors">
							<label class="form-check-label" for="tagColors">
								Use auto-generated tag colors
							</label>
						</div>
					</div>
					<div class="mb-3">
						<div class="form-check form-switch">
							<input class="form-check-input" type="checkbox" role="switch" id="htmlCheck"
								v-model="mailbox.showHTMLCheck">
							<label class="form-check-label" for="htmlCheck">
								Show HTML check message tab
							</label>
						</div>
					</div>
					<div class="mb-3">
						<div class="form-check form-switch">
							<input class="form-check-input" type="checkbox" role="switch" id="linkCheck"
								v-model="mailbox.showLinkCheck">
							<label class="form-check-label" for="linkCheck">
								Show link check message tab
							</label>
						</div>
					</div>
					<div class="mb-3" v-if="mailbox.uiConfig.SpamAssassin">
						<div class="form-check form-switch">
							<input class="form-check-input" type="checkbox" role="switch" id="spamCheck"
								v-model="mailbox.showSpamCheck">
							<label class="form-check-label" for="spamCheck">
								Show spam check message tab
							</label>
						</div>
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Close</button>
				</div>
			</div>
		</div>
	</div>
</template>
