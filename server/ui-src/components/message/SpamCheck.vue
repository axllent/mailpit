<script>
import { VcDonut } from "vue-css-donut-chart";
import axios from "axios";
import commonMixins from "../../mixins/CommonMixins";

export default {
	components: {
		VcDonut,
	},

	mixins: [commonMixins],

	props: {
		message: {
			type: Object,
			default: () => ({}),
		},
		// "spamassassin" or "rspamd"
		service: {
			type: String,
			default: "spamassassin",
		},
		// whether both services are enabled, so the title must identify the service
		showServiceName: {
			type: Boolean,
			default: false,
		},
	},

	emits: ["setSpamScore", "setBadgeStyle"],

	data() {
		return {
			error: false,
			check: false,
		};
	},

	computed: {
		isRspamd() {
			return this.service === "rspamd";
		},

		serviceName() {
			return this.isRspamd ? "Rspamd" : "SpamAssassin";
		},

		endpoint() {
			return this.isRspamd ? "rspamd-check" : "sa-check";
		},

		modalID() {
			return "AboutSpamAnalysis" + (this.isRspamd ? "Rspamd" : "");
		},

		// the score at which a message is considered spam
		threshold() {
			return this.check.Threshold > 0 ? this.check.Threshold : 5;
		},

		graphSections() {
			const score = this.check.Score;
			let p = Math.round((score / this.threshold) * 100);
			if (p > 100) {
				p = 100;
			} else if (p < 0) {
				p = 0;
			}

			let c = "#ffc107";
			if (this.check.IsSpam) {
				c = "#dc3545";
			}

			return [
				{
					label: score + " / " + this.threshold,
					value: p,
					color: c,
				},
			];
		},

		scoreColor() {
			return this.graphSections[0].color;
		},
	},

	watch: {
		message: {
			handler() {
				this.$emit("setSpamScore", false);
				this.doCheck();
			},
			deep: true,
		},
	},

	mounted() {
		this.doCheck();
	},

	methods: {
		doCheck() {
			this.check = false;

			// ignore any error, do not show loader
			axios
				.get(this.resolve("/api/v1/message/" + this.message.ID + "/" + this.endpoint), null)
				.then((result) => {
					this.check = result.data;
					this.error = false;
					this.setIcons();
				})
				.catch((error) => {
					// handle error
					if (error.response && error.response.data) {
						// The request was made and the server responded with a status code
						// that falls out of the range of 2xx
						if (error.response.data.Error) {
							this.error = error.response.data.Error;
						} else {
							this.error = error.response.data;
						}
					} else if (error.request) {
						// The request was made but no response was received
						// `error.request` is an instance of XMLHttpRequest in the browser and an instance of
						// http.ClientRequest in node.js
						this.error = "Error sending data to the server. Please try again.";
					} else {
						// Something happened in setting up the request that triggered an Error
						this.error = error.message;
					}
				});
		},

		badgeStyle(ignorePadding = false) {
			let badgeStyle = "bg-success";
			if (this.check.Error) {
				badgeStyle = "bg-warning text-primary";
			} else if (this.check.IsSpam) {
				badgeStyle = "bg-danger";
			} else if (this.check.Score >= this.threshold * 0.8) {
				badgeStyle = "bg-warning text-primary";
			}

			if (!ignorePadding && String(this.check.Score).includes(".")) {
				badgeStyle += " p-1";
			}

			return badgeStyle;
		},

		setIcons() {
			let score = this.check.Score;
			if (this.check.Error && this.check.Error !== "") {
				score = "!";
			}
			const badgeStyle = this.badgeStyle();
			this.$emit("setBadgeStyle", badgeStyle);
			this.$emit("setSpamScore", score);
		},
	},
};
</script>

<template>
	<div class="row mb-3 w-100 align-items-center">
		<div class="col">
			<h4 class="mb-0">
				Spam Analysis<span v-if="showServiceName"> ({{ serviceName }})</span>
			</h4>
		</div>
		<div class="col-auto">
			<button class="btn btn-outline-secondary" data-bs-toggle="modal" :data-bs-target="'#' + modalID">
				<i class="bi bi-info-circle-fill"></i>
				Help
			</button>
		</div>
	</div>

	<template v-if="error || check.Error != ''">
		<p>Your message could not be checked</p>
		<div v-if="error" class="alert alert-warning">
			{{ error }}
		</div>
		<div v-else class="alert alert-warning">
			There was an error contacting the configured {{ serviceName }} server: {{ check.Error }}
		</div>
	</template>

	<template v-else-if="check">
		<div class="row w-100 mt-5">
			<div class="col-xl-5 mb-2">
				<vc-donut
					:sections="graphSections"
					background="var(--bs-body-bg)"
					:size="230"
					unit="px"
					:thickness="20"
					:total="100"
					:start-angle="270"
					:auto-adjust-text-size="true"
					foreground="#198754"
				>
					<h2 class="m-0" :class="scoreColor" @click="scrollToWarnings">
						{{ check.Score }} / {{ threshold }}
					</h2>
					<div class="text-body mt-2">
						<span v-if="check.IsSpam" class="text-white badge rounded-pill bg-danger p-2">Spam</span>
						<span v-else class="badge rounded-pill p-2" :class="badgeStyle()">Not spam</span>
					</div>
				</vc-donut>
			</div>
			<div class="col-xl-7">
				<div class="row w-100 py-2 border-bottom">
					<div class="col-2 col-lg-1">
						<strong>Score</strong>
					</div>
					<div class="col-10 col-lg-5">
						<strong>Rule <span class="d-none d-lg-inline">name</span></strong>
					</div>
					<div class="col-auto d-none d-lg-block">
						<strong>Description</strong>
					</div>
				</div>

				<div v-for="r in check.Rules" :key="'rule_' + r.Name" class="row w-100 py-2 border-bottom small">
					<div class="col-2 col-lg-1">
						{{ r.Score }}
					</div>
					<div class="col-10 col-lg-5">
						{{ r.Name }}
					</div>
					<div class="col-auto col-lg-6 mt-2 mt-lg-0 offset-2 offset-lg-0">
						{{ r.Description }}
					</div>
				</div>
			</div>
		</div>
	</template>

	<div :id="modalID" class="modal fade" tabindex="-1" :aria-labelledby="modalID + 'Label'" aria-hidden="true">
		<div class="modal-dialog modal-lg modal-dialog-scrollable">
			<div class="modal-content">
				<div class="modal-header">
					<h1 :id="modalID + 'Label'" class="modal-title fs-5">About Spam Analysis</h1>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div class="modal-body">
					<div :id="modalID + 'Accordion'" class="accordion">
						<div class="accordion-item">
							<h2 class="accordion-header">
								<button
									class="accordion-button collapsed"
									type="button"
									data-bs-toggle="collapse"
									:data-bs-target="'#' + modalID + 'col1'"
									aria-expanded="false"
									:aria-controls="modalID + 'col1'"
								>
									What is Spam Analysis?
								</button>
							</h2>
							<div
								:id="modalID + 'col1'"
								class="accordion-collapse collapse"
								:data-bs-parent="'#' + modalID + 'Accordion'"
							>
								<div class="accordion-body">
									<p>
										Mailpit integrates with {{ serviceName }} to provide you with some insight into
										the "spamminess" of your messages. It sends your complete message (including any
										attachments) to a running {{ serviceName }} server and then displays the results
										returned by {{ serviceName }}.
									</p>
								</div>
							</div>
						</div>
						<div class="accordion-item">
							<h2 class="accordion-header">
								<button
									class="accordion-button collapsed"
									type="button"
									data-bs-toggle="collapse"
									:data-bs-target="'#' + modalID + 'col2'"
									aria-expanded="false"
									:aria-controls="modalID + 'col2'"
								>
									How does the point system work?
								</button>
							</h2>
							<div
								:id="modalID + 'col2'"
								class="accordion-collapse collapse"
								:data-bs-parent="'#' + modalID + 'Accordion'"
							>
								<div class="accordion-body">
									<p>
										<template v-if="isRspamd">
											Rspamd recommends an action (such as "add header" or "reject") based on the
											score. Mailpit treats the message as spam once the score reaches the
											<code>add header</code> threshold (currently <code>{{ threshold }}</code
											>).
										</template>
										<template v-else>
											The default spam threshold is <code>5</code>, meaning any score lower than 5
											is considered ham (not spam), and any score of 5 or above is spam.
										</template>
									</p>
									<p>
										{{ serviceName }} will also return the tests which are triggered by the message.
										These tests can differ depending on the configuration of your {{ serviceName }}
										server. The total of this score makes up the "spamminess" of the message.
									</p>
								</div>
							</div>
						</div>
						<div class="accordion-item">
							<h2 class="accordion-header">
								<button
									class="accordion-button collapsed"
									type="button"
									data-bs-toggle="collapse"
									:data-bs-target="'#' + modalID + 'col3'"
									aria-expanded="false"
									:aria-controls="modalID + 'col3'"
								>
									But I don't agree with the results...
								</button>
							</h2>
							<div
								:id="modalID + 'col3'"
								class="accordion-collapse collapse"
								:data-bs-parent="'#' + modalID + 'Accordion'"
							>
								<div class="accordion-body">
									<p>
										Mailpit does not manipulate the results nor determine the "spamminess" of your
										message. The result is what {{ serviceName }} returns, and it entirely dependent
										on how {{ serviceName }} is set up and optionally trained.
									</p>
									<p>
										This tool is simply provided as an aid to assist you. If you are running your
										own instance of {{ serviceName }}, then you look into your {{ serviceName }}
										configuration.
									</p>
								</div>
							</div>
						</div>
						<div class="accordion-item">
							<h2 class="accordion-header">
								<button
									class="accordion-button collapsed"
									type="button"
									data-bs-toggle="collapse"
									:data-bs-target="'#' + modalID + 'col4'"
									aria-expanded="false"
									:aria-controls="modalID + 'col4'"
								>
									Where can I find more information about the triggered rules?
								</button>
							</h2>
							<div
								:id="modalID + 'col4'"
								class="accordion-collapse collapse"
								:data-bs-parent="'#' + modalID + 'Accordion'"
							>
								<div class="accordion-body">
									<p v-if="isRspamd">
										Rspamd symbols are documented in the
										<a href="https://docs.rspamd.com/" target="_blank">Rspamd documentation</a>, and
										your own server may also include custom rules.
									</p>
									<p v-else>
										Unfortunately the current
										<a href="https://spamassassin.apache.org/" target="_blank"
											>SpamAssassin website</a
										>
										no longer contains any relative documentation about these, most likely because
										the rules come from different locations and change often. You will need to
										search the internet for these yourself.
									</p>
								</div>
							</div>
						</div>
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
				</div>
			</div>
		</div>
	</div>
</template>
