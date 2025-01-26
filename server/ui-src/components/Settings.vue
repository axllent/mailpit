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
			chaosConfig: false,
			chaosUpdated: false,
		}
	},

	watch: {
		theme(v) {
			if (v == 'auto') {
				localStorage.removeItem('theme')
			} else {
				localStorage.setItem('theme', v)
			}
			this.setTheme()
		},

		chaosConfig: {
			handler() {
				this.chaosUpdated = true
			},
			deep: true
		}
	},

	mounted() {
		this.setTheme()
		this.$nextTick(function () {
			Tags.init('select.tz')
		})
	},

	methods: {
		setTheme() {
			if (
				this.theme === 'auto' &&
				window.matchMedia('(prefers-color-scheme: dark)').matches
			) {
				document.documentElement.setAttribute('data-bs-theme', 'dark')
			} else {
				document.documentElement.setAttribute('data-bs-theme', this.theme)
			}
		},

		loadChaos() {
			this.get(this.resolve('/api/v1/chaos'), null, (response) => {
				this.chaosConfig = response.data
				this.$nextTick(() => {
					this.chaosUpdated = false
				})
			})
		},

		saveChaos() {
			this.put(this.resolve('/api/v1/chaos'), this.chaosConfig, (response) => {
				this.chaosConfig = response.data
				this.$nextTick(() => {
					this.chaosUpdated = false
				})
			})
		}
	}
}
</script>

<template>
	<div class="modal fade" id="SettingsModal" tabindex="-1" aria-labelledby="SettingsModalLabel" aria-hidden="true"
		data-bs-keyboard="false">
		<div class="modal-dialog modal-lg">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title" id="SettingsModalLabel">Mailpit settings</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div class="modal-body">
					<ul class="nav nav-tabs" id="myTab" role="tablist" v-if="mailbox.uiConfig.ChaosEnabled">
						<li class="nav-item" role="presentation">
							<button class="nav-link active" id="ui-tab" data-bs-toggle="tab"
								data-bs-target="#ui-tab-pane" type="button" role="tab" aria-controls="ui-tab-pane"
								aria-selected="true">Web UI</button>
						</li>
						<li class="nav-item" role="presentation">
							<button class="nav-link" id="chaos-tab" data-bs-toggle="tab"
								data-bs-target="#chaos-tab-pane" type="button" role="tab" aria-controls="chaos-tab-pane"
								aria-selected="false" @click="loadChaos">Chaos</button>
						</li>
					</ul>

					<div class="tab-content">
						<div class="tab-pane fade show active" id="ui-tab-pane" role="tabpanel" aria-labelledby="ui-tab"
							tabindex="0">
							<div class="my-3">
								<label for="theme" class="form-label">Mailpit theme</label>
								<select class="form-select" v-model="theme" id="theme">
									<option value="auto">Auto (detect from browser)</option>
									<option value="light">Light theme</option>
									<option value="dark">Dark theme</option>
								</select>
							</div>
							<div class="mb-3">
								<label for="timezone" class="form-label">Timezone (for date searches)</label>
								<select class="form-select tz" v-model="mailbox.timeZone" id="timezone"
									data-allow-same="true">
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

						<div class="tab-pane fade" id="chaos-tab-pane" role="tabpanel" aria-labelledby="chaos-tab"
							tabindex="0" v-if="mailbox.uiConfig.ChaosEnabled">
							<p class="my-3">
								<b>Chaos</b> allows you to set random SMTP failures and response codes at various
								stages in a SMTP transaction to test application resilience
								(<a href="https://mailpit.axllent.org/docs/integration/chaos/" target="_blank">
									see documentation
								</a>).
							</p>

							<ul>
								<li>
									<code>Response code</code> is the SMTP error code returned by the server if this
									error is triggered. Error codes must range between 400 and 599.
								</li>
								<li>
									<code>Error probability</code> is the % chance that the error will occur per message
									delivery, where <code>0</code>(%) is disabled and <code>100</code>(%) wil always
									trigger. A probability of <code>50</code> will trigger on approximately 50% of
									messages received.
								</li>
							</ul>

							<template v-if="chaosConfig">
								<div class="mt-4 mb-4" :class="chaosUpdated ? 'was-validated' : ''">
									<div class="mb-4">
										<label>Trigger: <code>Sender</code></label>
										<div class="form-text">
											Trigger an error response based on the sender (From / Sender).
										</div>
										<div class="row mt-1">
											<div class="col">
												<label class="form-label">
													Response code
												</label>
												<input type="number" class="form-control"
													v-model.number="chaosConfig.Sender.ErrorCode" min="400" max="599"
													required>
											</div>
											<div class="col">
												<label class="form-label">
													Error probability ({{ chaosConfig.Sender.Probability }}%)
												</label>
												<input type="range" class="form-range mt-1" min="0" max="100"
													v-model.number="chaosConfig.Sender.Probability">
											</div>
										</div>
									</div>

									<div class="mb-4">
										<label>Trigger: <code>Recipient</code></label>
										<div class="form-text">
											Trigger an error response based on the recipients (To, Cc, Bcc).
										</div>
										<div class="row mt-1">
											<div class="col">
												<label class="form-label">
													Response code
												</label>
												<input type="number" class="form-control"
													v-model.number="chaosConfig.Recipient.ErrorCode" min="400" max="599"
													required>
											</div>
											<div class="col">
												<label class="form-label">
													Error probability ({{ chaosConfig.Recipient.Probability }}%)
												</label>
												<input type="range" class="form-range mt-1" min="0" max="100"
													v-model.number="chaosConfig.Recipient.Probability">
											</div>
										</div>
									</div>

									<div class="mb-4">
										<label>Trigger: <code>Authentication</code></label>
										<div class="form-text">
											Trigger an authentication error response.
											Note that SMTP authentication must be configured too.
										</div>
										<div class="row mt-1">
											<div class="col">
												<label class="form-label">
													Response code
												</label>
												<input type="number" class="form-control"
													v-model.number="chaosConfig.Authentication.ErrorCode" min="400"
													max="599" required>
											</div>
											<div class="col">
												<label class="form-label">
													Error probability ({{ chaosConfig.Authentication.Probability }}%)
												</label>
												<input type="range" class="form-range mt-1" min="0" max="100"
													v-model.number="chaosConfig.Authentication.Probability">
											</div>
										</div>
									</div>
								</div>

								<div v-if="chaosUpdated" class="mb-3 text-center">
									<button class="btn btn-success" @click="saveChaos">Update Chaos</button>
								</div>
							</template>
						</div>

					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Close</button>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>
