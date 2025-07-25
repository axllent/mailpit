<script>
import { VcDonut } from "vue-css-donut-chart";
import axios from "axios";
import commonMixins from "../../mixins/CommonMixins";
import { Tooltip } from "bootstrap";
import DOMPurify from "dompurify";

export default {
	components: {
		VcDonut,
	},

	mixins: [commonMixins],

	props: {
		message: {
			type: Object,
			required: true,
		},
	},

	emits: ["setHtmlScore", "setBadgeStyle"],

	data() {
		return {
			error: false,
			check: false,
			platforms: [],
			allPlatforms: {
				windows: "Windows",
				"windows-mail": "Windows Mail",
				"outlook-com": "Outlook.com",
				macos: "macOS",
				ios: "iOS",
				android: "Android",
				"desktop-webmail": "Desktop Webmail",
				"mobile-webmail": "Mobile Webmail",
			},
		};
	},

	computed: {
		summary() {
			if (!this.check) {
				return false;
			}

			const result = {
				Warnings: [],
				Total: {
					Nodes: this.check.Total.Nodes,
				},
			};

			for (let i = 0; i < this.check.Warnings.length; i++) {
				const o = JSON.parse(JSON.stringify(this.check.Warnings[i]));

				// for <script> test
				if (o.Results.length === 0) {
					result.Warnings.push(o);
					continue;
				}

				// filter by enabled platforms
				const results = o.Results.filter((w) => {
					return this.platforms.indexOf(w.Platform) !== -1;
				});

				if (results.length === 0) {
					continue;
				}

				// recalculate the percentages
				let y = 0;
				let p = 0;
				let n = 0;

				results.forEach((r) => {
					if (r.Support === "yes") {
						y++;
					} else if (r.Support === "partial") {
						p++;
					} else {
						n++;
					}
				});
				const total = y + p + n;
				o.Results = results;
				o.Score = {
					Found: o.Score.Found,
					Supported: (y / total) * 100,
					Partial: (p / total) * 100,
					Unsupported: (n / total) * 100,
				};

				result.Warnings.push(o);
			}

			let maxPartial = 0;
			let maxUnsupported = 0;
			result.Warnings.forEach((w) => {
				let scoreWeight = 1;
				if (w.Score.Found < result.Total.Nodes) {
					// each error is weighted based on the number of occurrences vs: the total message nodes
					scoreWeight = w.Score.Found / result.Total.Nodes;
				}

				// pseudo-classes & at-rules need to be weighted lower as we do not know how many times they
				// are actually used in the HTML, and including things like bootstrap styles completely throws
				// off the calculation as these dominate.
				if (this.isPseudoClassOrAtRule(w.Title)) {
					scoreWeight = 0.05;
					w.PseudoClassOrAtRule = true;
				}

				const scorePartial = w.Score.Partial * scoreWeight;
				const scoreUnsupported = w.Score.Unsupported * scoreWeight;
				if (scorePartial > maxPartial) {
					maxPartial = scorePartial;
				}
				if (scoreUnsupported > maxUnsupported) {
					maxUnsupported = scoreUnsupported;
				}
			});

			// sort warnings by final score
			result.Warnings.sort((a, b) => {
				let aWeight =
					a.Score.Found > result.Total.Nodes ? result.Total.Nodes : a.Score.Found / result.Total.Nodes;
				let bWeight =
					b.Score.Found > result.Total.Nodes ? result.Total.Nodes : b.Score.Found / result.Total.Nodes;

				if (this.isPseudoClassOrAtRule(a.Title)) {
					aWeight = 0.05;
				}

				if (this.isPseudoClassOrAtRule(b.Title)) {
					bWeight = 0.05;
				}

				return (
					(a.Score.Unsupported + a.Score.Partial) * aWeight <
					(b.Score.Unsupported + b.Score.Partial) * bWeight
				);
			});

			result.Total.Supported = 100 - maxPartial - maxUnsupported;
			result.Total.Partial = maxPartial;
			result.Total.Unsupported = maxUnsupported;

			this.$emit("setHtmlScore", result.Total.Supported);

			return result;
		},

		graphSections() {
			const s = Math.round(this.summary.Total.Supported);
			const p = Math.round(this.summary.Total.Partial);
			const u = 100 - s - p;
			return [
				{
					label: this.round2dm(this.summary.Total.Supported) + "% supported",
					value: s,
					color: "#198754",
				},
				{
					label: this.round2dm(this.summary.Total.Partial) + "% partially supported",
					value: p,
					color: "#ffc107",
				},
				{
					label: this.round2dm(this.summary.Total.Unsupported) + "% not supported",
					value: u,
					color: "#dc3545",
				},
			];
		},

		// colors depend on both varying unsupported & partially unsupported percentages
		scoreColor() {
			if (this.summary.Total.Unsupported < 5 && this.summary.Total.Partial < 10) {
				this.$emit("setBadgeStyle", "bg-success");
				return "text-success";
			} else if (this.summary.Total.Unsupported < 10 && this.summary.Total.Partial < 15) {
				this.$emit("setBadgeStyle", "bg-warning text-primary");
				return "text-warning";
			}

			this.$emit("setBadgeStyle", "bg-danger");
			return "text-danger";
		},
	},

	watch: {
		message: {
			handler() {
				this.$emit("setHtmlScore", false);
				this.doCheck();
			},
			deep: true,
		},
		platforms(v) {
			localStorage.setItem("html-check-platforms", JSON.stringify(v));
		},
	},

	mounted() {
		this.loadConfig();
		this.doCheck();
	},

	methods: {
		doCheck() {
			this.check = false;

			if (this.message.HTML === "") {
				return;
			}

			// ignore any error, do not show loader
			axios
				.get(this.resolve("/api/v1/message/" + this.message.ID + "/html-check"), null)
				.then((result) => {
					this.check = result.data;
					this.error = false;

					// set tooltips
					window.setTimeout(() => {
						const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]');
						[...tooltipTriggerList].map((tooltipTriggerEl) => new Tooltip(tooltipTriggerEl));
					}, 500);
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

		loadConfig() {
			const platforms = localStorage.getItem("html-check-platforms");
			if (platforms) {
				try {
					this.platforms = JSON.parse(platforms);
				} catch {
					// if parsing fails, reset to default
					this.platforms = [];
				}
			}

			// set all options
			if (this.platforms.length === 0) {
				this.platforms = Object.keys(this.allPlatforms);
			}
		},

		// return a platform's families (email clients)
		families(k) {
			if (this.check.Platforms[k]) {
				return this.check.Platforms[k];
			}

			return [];
		},

		// return whether the test string is a pseudo class (:<test>) or at rule (@<test>)
		isPseudoClassOrAtRule(t) {
			return t.match(/^(:|@)/);
		},

		round(v) {
			return Math.round(v);
		},

		round2dm(v) {
			return Math.round(v * 100) / 100;
		},

		scrollToWarnings() {
			if (!this.$refs.warnings) {
				return;
			}

			this.$refs.warnings.scrollIntoView({ behavior: "smooth" });
		},

		// Sanitize HTML to prevent XSS
		sanitizeHTML(html) {
			return DOMPurify.sanitize(html);
		},
	},
};
</script>

<template>
	<template v-if="error">
		<p>HTML check failed to load:</p>
		<div class="alert alert-warning">
			{{ error }}
		</div>
	</template>

	<template v-if="summary">
		<div class="mt-5 mb-3">
			<div class="row w-100">
				<div class="col-md-8">
					<vc-donut
						:sections="graphSections"
						background="var(--bs-body-bg)"
						:size="180"
						unit="px"
						:thickness="20"
						has-legend
						legend-placement="bottom"
						:total="100"
						:start-angle="0"
						:auto-adjust-text-size="true"
						@section-click="scrollToWarnings"
					>
						<h2 class="m-0" :class="scoreColor" @click="scrollToWarnings">
							{{ round2dm(summary.Total.Supported) }}%
						</h2>
						<div class="text-body">support</div>
						<template #legend>
							<p class="my-3 small mb-1 text-center" @click="scrollToWarnings">
								<span class="text-nowrap">
									<i class="bi bi-circle-fill text-success"></i>
									{{ round2dm(summary.Total.Supported) }}% supported
								</span>
								&nbsp;
								<span class="text-nowrap">
									<i class="bi bi-circle-fill text-warning"></i>
									{{ round2dm(summary.Total.Partial) }}% partially supported
								</span>
								&nbsp;
								<span class="text-nowrap">
									<i class="bi bi-circle-fill text-danger"></i>
									{{ round2dm(summary.Total.Unsupported) }}% not supported
								</span>
							</p>
							<p class="small text-muted">calculated from {{ formatNumber(check.Total.Tests) }} tests</p>
						</template>
					</vc-donut>

					<div class="input-group justify-content-center mb-3">
						<button
							class="btn btn-outline-secondary"
							data-bs-toggle="modal"
							data-bs-target="#AboutHTMLCheckResults"
						>
							<i class="bi bi-info-circle-fill"></i>
							Help
						</button>
					</div>
				</div>
				<div class="col-md">
					<h2 class="h5 mb-3">Tested platforms:</h2>
					<div v-for="(p, k) in allPlatforms" :key="'check_' + k" class="form-check form-switch">
						<input
							:id="'Check_' + k"
							v-model="platforms"
							class="form-check-input"
							type="checkbox"
							role="switch"
							:value="k"
							:aria-label="p"
						/>
						<label
							class="form-check-label"
							:for="'Check_' + k"
							:class="platforms.indexOf(k) !== -1 ? '' : 'text-muted'"
							:title="families(k).join(', ')"
							data-bs-toggle="tooltip"
							:data-bs-title="families(k).join(', ')"
						>
							{{ p }}
						</label>
					</div>
				</div>
			</div>
		</div>

		<template v-if="summary.Warnings.length">
			<h4 ref="warnings" class="h5 mt-4">
				{{ summary.Warnings.length }} Warnings from {{ formatNumber(summary.Total.Nodes) }} HTML nodes:
			</h4>
			<div id="warnings" class="accordion">
				<div v-for="(warning, i) in summary.Warnings" :key="'warning_' + i" class="accordion-item">
					<h2 class="accordion-header">
						<button
							class="accordion-button collapsed"
							type="button"
							data-bs-toggle="collapse"
							:data-bs-target="'#' + warning.Slug"
							aria-expanded="false"
							:aria-controls="warning.Slug"
						>
							<div class="row w-100 w-lg-75">
								<div class="col-sm">
									{{ warning.Title }}
									<span class="ms-2 small badge text-bg-secondary" title="Test category">
										{{ warning.Category }}
									</span>
									<span
										class="ms-2 small badge text-bg-light"
										title="The number of times this was detected"
									>
										x {{ warning.Score.Found }}
									</span>
								</div>
								<div class="col-sm mt-2 mt-sm-0">
									<div class="progress-stacked">
										<div
											class="progress"
											role="progressbar"
											aria-label="Supported"
											:aria-valuenow="warning.Score.Supported"
											aria-valuemin="0"
											aria-valuemax="100"
											:style="{ width: warning.Score.Supported + '%' }"
											title="Supported"
										>
											<div class="progress-bar bg-success">
												{{ round(warning.Score.Supported) + "%" }}
											</div>
										</div>
										<div
											class="progress"
											role="progressbar"
											aria-label="Partial"
											:aria-valuenow="warning.Score.Partial"
											aria-valuemin="0"
											aria-valuemax="100"
											:style="{ width: warning.Score.Partial + '%' }"
											title="Partial support"
										>
											<div class="progress-bar progress-bar-striped bg-warning text-dark">
												{{ round(warning.Score.Partial) + "%" }}
											</div>
										</div>
										<div
											class="progress"
											role="progressbar"
											aria-label="No"
											:aria-valuenow="warning.Score.Unsupported"
											aria-valuemin="0"
											aria-valuemax="100"
											:style="{ width: warning.Score.Unsupported + '%' }"
											title="Not supported"
										>
											<div class="progress-bar bg-danger">
												{{ round(warning.Score.Unsupported) + "%" }}
											</div>
										</div>
									</div>
								</div>
							</div>
						</button>
					</h2>
					<div :id="warning.Slug" class="accordion-collapse collapse" data-bs-parent="#warnings">
						<div class="accordion-body">
							<p v-if="warning.Description !== '' || warning.PseudoClassOrAtRule">
								<span v-if="warning.PseudoClassOrAtRule" class="d-block alert alert-warning mb-2">
									<i class="bi bi-info-circle me-2"></i>
									Detected {{ warning.Score.Found }} <code>{{ warning.Title }}</code>
									<template v-if="warning.Score.Found === 1">property</template>
									<template v-else>properties</template>
									in the CSS styles, but unable to test if used or not.
								</span>
								<!-- eslint-disable vue/no-v-html -->
								<span
									v-if="warning.Description !== ''"
									class="me-2"
									v-html="sanitizeHTML(warning.Description)"
								></span>
								<!-- -eslint-disable vue/no-v-html -->
							</p>

							<template v-if="warning.Results.length">
								<h3 class="h6">Clients with partial or no support:</h3>
								<p>
									<small
										v-for="(warningRes, wi) in warning.Results"
										:key="'warning_results_' + wi"
										class="text-nowrap d-inline-block me-4"
									>
										<i
											class="bi bi-circle-fill"
											:class="warningRes.Support === 'no' ? 'text-danger' : 'text-warning'"
											:title="
												warningRes.Support === 'no' ? 'Not supported' : 'Partially supported'
											"
										></i>
										{{ warningRes.Name }}
										<span
											v-if="warningRes.NoteNumber !== ''"
											class="badge text-bg-secondary"
											title="See notes"
										>
											{{ warningRes.NoteNumber }}
										</span>
									</small>
								</p>
							</template>

							<div v-if="Object.keys(warning.NotesByNumber).length" class="mt-3">
								<h3 class="h6">Notes:</h3>
								<div
									v-for="(n, ni) in warning.NotesByNumber"
									:key="'warning_notes' + ni"
									class="small row my-2"
								>
									<div class="col-auto pe-0">
										<span class="badge text-bg-secondary">
											{{ ni }}
										</span>
									</div>
									<div class="col" v-html="sanitizeHTML(n)"></div>
								</div>
							</div>

							<p v-if="warning.URL" class="small mt-3 mb-0">
								<a :href="warning.URL" target="_blank">Online reference</a>
							</p>
						</div>
					</div>
				</div>
			</div>

			<p class="text-center text-muted small mt-4">
				Scores based on <b>{{ check.Total.Tests }}</b> tests of HTML and CSS properties using compatibility data
				from <a href="https://www.caniemail.com/" target="_blank">caniemail.com</a>.
			</p>
		</template>

		<div
			id="AboutHTMLCheckResults"
			class="modal fade"
			tabindex="-1"
			aria-labelledby="AboutHTMLCheckResultsLabel"
			aria-hidden="true"
		>
			<div class="modal-dialog modal-lg modal-dialog-scrollable">
				<div class="modal-content">
					<div class="modal-header">
						<h1 id="AboutHTMLCheckResultsLabel" class="modal-title fs-5">About HTML check</h1>
						<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
					</div>
					<div class="modal-body">
						<div id="HTMLCheckAboutAccordion" class="accordion">
							<div class="accordion-item">
								<h2 class="accordion-header">
									<button
										class="accordion-button collapsed"
										type="button"
										data-bs-toggle="collapse"
										data-bs-target="#col1"
										aria-expanded="false"
										aria-controls="col1"
									>
										What is HTML check?
									</button>
								</h2>
								<div
									id="col1"
									class="accordion-collapse collapse"
									data-bs-parent="#HTMLCheckAboutAccordion"
								>
									<div class="accordion-body">
										The support for HTML/CSS messages varies greatly across email clients. HTML
										check attempts to calculate the overall support for your email for all selected
										platforms to give you some idea of the general compatibility of your HTML email.
									</div>
								</div>
							</div>
							<div class="accordion-item">
								<h2 class="accordion-header">
									<button
										class="accordion-button collapsed"
										type="button"
										data-bs-toggle="collapse"
										data-bs-target="#col2"
										aria-expanded="false"
										aria-controls="col2"
									>
										How does it work?
									</button>
								</h2>
								<div
									id="col2"
									class="accordion-collapse collapse"
									data-bs-parent="#HTMLCheckAboutAccordion"
								>
									<div class="accordion-body">
										<p>
											Internally the original HTML message is run against
											<b>{{ check.Total.Tests }}</b> different HTML and CSS tests. All tests
											(except for <code>&lt;script&gt;</code>) correspond to a test on
											<a href="https://www.caniemail.com/" target="_blank">caniemail.com</a>, and
											the final score is calculated using the available compatibility data.
										</p>
										<p>
											CSS support is very difficult to programmatically test, especially if a
											message contains CSS style blocks or is linked to remote stylesheets. Remote
											stylesheets are, unless blocked via
											<code>--block-remote-css-and-fonts</code>, downloaded and injected into the
											message as style blocks. The email is then
											<a href="https://github.com/vanng822/go-premailer" target="_blank"
												>inlined</a
											>
											to matching HTML elements. This gives Mailpit fairly accurate results.
										</p>
										<p>
											CSS properties such as <code>@font-face</code>, <code>:visited</code>,
											<code>:hover</code> etc cannot be inlined however, so these are searched for
											within CSS blocks. This method is not accurate as Mailpit does not know how
											many nodes it actually applies to, if any, so they are weighted lightly (5%)
											as not to affect the score. An example of this would be any email linking to
											the full bootstrap CSS which contains dozens of unused attributes.
										</p>
										<p>
											All warnings are displayed with their respective support, including any
											specific notes, and it is up to you to decide what you do with that
											information and how badly it may impact your message.
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
										data-bs-target="#col3"
										aria-expanded="false"
										aria-controls="col3"
									>
										Is the final score accurate?
									</button>
								</h2>
								<div
									id="col3"
									class="accordion-collapse collapse"
									data-bs-parent="#HTMLCheckAboutAccordion"
								>
									<div class="accordion-body">
										<p>
											There are many ways to define "accurate", and how one should calculate the
											compatibility score of an email. There is also no way to programmatically
											determine the relevance of a single test to the entire email.
										</p>
										<p>
											For each test, Mailpit calculates both the unsupported & partially-supported
											percentages in relation to the number of matches against the total number of
											nodes (elements) in the HTML. The maximum unsupported and
											partially-supported weighted scores are then used for the final score (ie:
											worst case scenario).
										</p>
										<p>
											To try explain this logic in very simple terms: Assuming a
											<code>&lt;script&gt;</code> node (element) has 100% failure (not supported
											in any email client), and a <code>&lt;p&gt;</code> node has 100% pass
											(supported).
										</p>
										<ul>
											<li>
												An email containing just a single <code>&lt;script&gt;</code>: the final
												score is 0% supported.
											</li>
											<li>
												An email containing just a <code>&lt;script&gt;</code> and a
												<code>&lt;p&gt;</code>: the final score is 50% supported.
											</li>
											<li>
												An email containing just a <code>&lt;script&gt;</code> and two
												<code>&lt;p&gt;</code>: the final score is 66.67% supported.
											</li>
										</ul>
										<p>
											Mailpit will sort the warnings according to their weighted unsupported
											scores.
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
										data-bs-target="#col4"
										aria-expanded="false"
										aria-controls="col4"
									>
										What about invalid HTML?
									</button>
								</h2>
								<div
									id="col4"
									class="accordion-collapse collapse"
									data-bs-parent="#HTMLCheckAboutAccordion"
								>
									<div class="accordion-body">
										HTML check does not detect if the original HTML is valid. In order to detect
										applied styles to every node, the HTML email is run through a parser which is
										very good at turning invalid input into valid output. It is what it is...
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
</template>
