<script>
import CommonMixins from "../mixins/CommonMixins";
import Tags from "bootstrap5-tags";
import timezones from "timezones-list";
import { mailbox } from "../stores/mailbox";

export default {
	mixins: [CommonMixins],

	data() {
		return {
			mailbox,
			theme: localStorage.getItem("theme") ? localStorage.getItem("theme") : "auto",
			timezones,
			chaosConfig: false,
			chaosUpdated: false,
			defaultReleaseAddressesOptions: localStorage.getItem("defaultReleaseAddresses")
				? JSON.parse(localStorage.getItem("defaultReleaseAddresses"))
				: [], // set with default release addresses
		};
	},

	watch: {
		theme(v) {
			if (v === "auto") {
				localStorage.removeItem("theme");
			} else {
				localStorage.setItem("theme", v);
			}
			this.setTheme();
		},

		chaosConfig: {
			handler() {
				this.chaosUpdated = true;
			},
			deep: true,
		},

		"mailbox.skipConfirmations"(v) {
			if (v) {
				localStorage.setItem("skip-confirmations", "true");
			} else {
				localStorage.removeItem("skip-confirmations");
			}
		},
	},

	mounted() {
		this.setTheme();
		this.$nextTick(() => {
			Tags.init("select.tz");
		});

		mailbox.skipConfirmations = localStorage.getItem("skip-confirmations");

		Tags.init("select.default-release-addresses");
	},

	methods: {
		setTheme() {
			if (this.theme === "auto" && window.matchMedia("(prefers-color-scheme: dark)").matches) {
				document.documentElement.setAttribute("data-bs-theme", "dark");
			} else {
				document.documentElement.setAttribute("data-bs-theme", this.theme);
			}
		},

		loadChaos() {
			this.get(this.resolve("/api/v1/chaos"), null, (response) => {
				this.chaosConfig = response.data;
				this.$nextTick(() => {
					this.chaosUpdated = false;
				});
			});
		},

		saveChaos() {
			this.put(this.resolve("/api/v1/chaos"), this.chaosConfig, (response) => {
				this.chaosConfig = response.data;
				this.$nextTick(() => {
					this.chaosUpdated = false;
				});
			});
		},
	},
};
</script>

<template>
	<div
		id="SettingsModal"
		class="modal fade"
		tabindex="-1"
		aria-labelledby="SettingsModalLabel"
		aria-hidden="true"
		data-bs-keyboard="false"
	>
		<div class="modal-dialog modal-lg">
			<div class="modal-content">
				<div class="modal-header">
					<h5 id="SettingsModalLabel" class="modal-title">Mailpit settings</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div class="modal-body">
					<ul id="myTab" class="nav nav-tabs" role="tablist">
						<li class="nav-item" role="presentation">
							<button
								id="ui-tab"
								class="nav-link active"
								data-bs-toggle="tab"
								data-bs-target="#ui-tab-pane"
								type="button"
								role="tab"
								aria-controls="ui-tab-pane"
								aria-selected="true"
							>
								Web UI
							</button>
						</li>
						<li
							v-if="mailbox.uiConfig.MessageRelay && mailbox.uiConfig.MessageRelay.Enabled"
							class="nav-item"
							role="presentation"
						>
							<button
								id="relay-tab"
								class="nav-link"
								data-bs-toggle="tab"
								data-bs-target="#relay-tab-pane"
								type="button"
								role="tab"
								aria-controls="relay-tab-pane"
								aria-selected="false"
							>
								Message release
							</button>
						</li>
						<li v-if="mailbox.uiConfig.ChaosEnabled" class="nav-item" role="presentation">
							<button
								id="chaos-tab"
								class="nav-link"
								data-bs-toggle="tab"
								data-bs-target="#chaos-tab-pane"
								type="button"
								role="tab"
								aria-controls="chaos-tab-pane"
								aria-selected="false"
								@click="loadChaos"
							>
								Chaos
							</button>
						</li>
					</ul>

					<div class="tab-content">
						<div
							id="ui-tab-pane"
							class="tab-pane fade show active"
							role="tabpanel"
							aria-labelledby="ui-tab"
							tabindex="0"
						>
							<div class="my-3">
								<label for="theme" class="form-label">Mailpit theme</label>
								<select id="theme" v-model="theme" class="form-select">
									<option value="auto">Auto (detect from browser)</option>
									<option value="light">Light theme</option>
									<option value="dark">Dark theme</option>
								</select>
							</div>
							<div class="mb-3">
								<label for="timezone" class="form-label">Timezone (for date searches)</label>
								<select
									id="timezone"
									v-model="mailbox.timeZone"
									class="form-select tz"
									data-allow-same="true"
								>
									<option disabled hidden value="">Select a timezone...</option>
									<option v-for="t in timezones" :key="t" :value="t.tzCode">{{ t.label }}</option>
								</select>
							</div>
							<div class="mb-3">
								<div class="form-check form-switch">
									<input
										id="tagColors"
										v-model="mailbox.showTagColors"
										class="form-check-input"
										type="checkbox"
										role="switch"
									/>
									<label class="form-check-label" for="tagColors">
										Use auto-generated tag colors
									</label>
								</div>
							</div>
							<div class="mb-3">
								<div class="form-check form-switch">
									<input
										id="htmlCheck"
										v-model="mailbox.showHTMLCheck"
										class="form-check-input"
										type="checkbox"
										role="switch"
									/>
									<label class="form-check-label" for="htmlCheck">
										Show HTML check message tab
									</label>
								</div>
							</div>
							<div class="mb-3">
								<div class="form-check form-switch">
									<input
										id="linkCheck"
										v-model="mailbox.showLinkCheck"
										class="form-check-input"
										type="checkbox"
										role="switch"
									/>
									<label class="form-check-label" for="linkCheck">
										Show link check message tab
									</label>
								</div>
							</div>
							<div v-if="mailbox.uiConfig.SpamAssassin" class="mb-3">
								<div class="form-check form-switch">
									<input
										id="spamCheck"
										v-model="mailbox.showSpamCheck"
										class="form-check-input"
										type="checkbox"
										role="switch"
									/>
									<label class="form-check-label" for="spamCheck">
										Show spam check message tab
									</label>
								</div>
							</div>
							<div class="mb-3">
								<div class="form-check form-switch">
									<input
										id="skip-confirmations"
										v-model="mailbox.skipConfirmations"
										class="form-check-input"
										type="checkbox"
										role="switch"
									/>
									<label class="form-check-label" for="skip-confirmations">
										Skip
										<template v-if="!mailbox.uiConfig.HideDeleteAllButton">
											<code>Delete all</code> &amp;
										</template>
										<code>Mark all read</code> confirmation dialogs
									</label>
								</div>
							</div>
						</div>

						<!-- Default relay addresses -->
						<div
							v-if="mailbox.uiConfig.MessageRelay && mailbox.uiConfig.MessageRelay.Enabled"
							id="relay-tab-pane"
							class="tab-pane fade"
							role="tabpanel"
							aria-labelledby="relay-tab"
							tabindex="0"
						>
							<div class="my-3 mb-5">
								<label class="form-label">Default release address(es)</label>
								<div class="form-text mb-2">
									You can designate the default "send to" addresses here, which will automatically
									populate the field in the message release dialog. This setting applies only to your
									browser. If this field is left empty, it will revert to the original recipients of
									the message.
								</div>
								<select
									v-model="mailbox.defaultReleaseAddresses"
									class="form-select tag-selector default-release-addresses"
									multiple
									data-allow-new="true"
									data-clear-end="true"
									data-allow-clear="true"
									data-placeholder="Enter email addresses..."
									data-add-on-blur="true"
									data-badge-style="primary"
									data-regex='^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|.(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$'
									data-separator="|,|"
								>
									<option value="">Enter email addresses...</option>
									<!-- you need at least one option with the placeholder -->
									<option
										v-for="t in defaultReleaseAddressesOptions"
										:key="'address+' + t"
										:value="t"
									>
										{{ t }}
									</option>
								</select>
								<div class="invalid-feedback">Invalid email address</div>
							</div>
						</div>

						<div
							v-if="mailbox.uiConfig.ChaosEnabled"
							id="chaos-tab-pane"
							class="tab-pane fade"
							role="tabpanel"
							aria-labelledby="chaos-tab"
							tabindex="0"
						>
							<p class="my-3">
								<b>Chaos</b> allows you to set random SMTP failures and response codes at various stages
								in a SMTP transaction to test application resilience (<a
									href="https://mailpit.axllent.org/docs/integration/chaos/"
									target="_blank"
								>
									see documentation </a
								>).
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
												<label class="form-label"> Response code </label>
												<input
													v-model.number="chaosConfig.Sender.ErrorCode"
													type="number"
													class="form-control"
													min="400"
													max="599"
													required
												/>
											</div>
											<div class="col">
												<label class="form-label">
													Error probability ({{ chaosConfig.Sender.Probability }}%)
												</label>
												<input
													v-model.number="chaosConfig.Sender.Probability"
													type="range"
													class="form-range mt-1"
													min="0"
													max="100"
												/>
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
												<label class="form-label"> Response code </label>
												<input
													v-model.number="chaosConfig.Recipient.ErrorCode"
													type="number"
													class="form-control"
													min="400"
													max="599"
													required
												/>
											</div>
											<div class="col">
												<label class="form-label">
													Error probability ({{ chaosConfig.Recipient.Probability }}%)
												</label>
												<input
													v-model.number="chaosConfig.Recipient.Probability"
													type="range"
													class="form-range mt-1"
													min="0"
													max="100"
												/>
											</div>
										</div>
									</div>

									<div class="mb-4">
										<label>Trigger: <code>Authentication</code></label>
										<div class="form-text">
											Trigger an authentication error response. Note that SMTP authentication must
											be configured too.
										</div>
										<div class="row mt-1">
											<div class="col">
												<label class="form-label"> Response code </label>
												<input
													v-model.number="chaosConfig.Authentication.ErrorCode"
													type="number"
													class="form-control"
													min="400"
													max="599"
													required
												/>
											</div>
											<div class="col">
												<label class="form-label">
													Error probability ({{ chaosConfig.Authentication.Probability }}%)
												</label>
												<input
													v-model.number="chaosConfig.Authentication.Probability"
													type="range"
													class="form-range mt-1"
													min="0"
													max="100"
												/>
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
