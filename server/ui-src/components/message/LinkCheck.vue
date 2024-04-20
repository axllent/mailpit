<script>
import axios from 'axios'
import commonMixins from '../../mixins/CommonMixins'

export default {
	props: {
		message: Object,
	},

	emits: ["setLinkErrors"],

	mixins: [commonMixins],

	data() {
		return {
			error: false,
			autoScan: false,
			followRedirects: false,
			check: false,
			loaded: false,
			loading: false,
		}
	},

	created() {
		this.autoScan = localStorage.getItem('LinkCheckAutoScan')
		this.followRedirects = localStorage.getItem('LinkCheckFollowRedirects')
	},

	mounted() {
		this.loaded = true
		if (this.autoScan) {
			this.doCheck()
		}
	},

	watch: {
		autoScan(v) {
			if (!this.loaded) {
				return
			}
			if (v) {
				localStorage.setItem('LinkCheckAutoScan', true)
				if (!this.check) {
					this.doCheck()
				}
			} else {
				localStorage.removeItem('LinkCheckAutoScan')
			}
		},
		followRedirects(v) {
			if (!this.loaded) {
				return
			}
			if (v) {
				localStorage.setItem('LinkCheckFollowRedirects', true)
			} else {
				localStorage.removeItem('LinkCheckFollowRedirects')
			}
			if (this.check) {
				this.doCheck()
			}
		}
	},

	computed: {
		groupedStatuses: function () {
			let results = {}

			if (!this.check) {
				return results
			}

			// group by status
			this.check.Links.forEach(function (r) {
				if (!results[r.StatusCode]) {
					let css = ""
					if (r.StatusCode >= 400 || r.StatusCode === 0) {
						css = "text-danger"
					} else if (r.StatusCode >= 300) {
						css = "text-info"
					}

					if (r.StatusCode === 0) {
						r.Status = 'Cannot connect to server'
					}
					results[r.StatusCode] = {
						StatusCode: r.StatusCode,
						Status: r.Status,
						Class: css,
						URLS: []
					}
				}
				results[r.StatusCode].URLS.push(r.URL)
			})

			let newArr = []

			for (const i in results) {
				newArr.push(results[i])
			}

			// sort statuses
			let sorted = newArr.sort((a, b) => {
				if (a.StatusCode === 0) {
					return false
				}
				return a.StatusCode < b.StatusCode
			})


			return sorted
		}
	},

	methods: {
		doCheck: function () {
			this.check = false
			this.loading = true
			let uri = this.resolve('/api/v1/message/' + this.message.ID + '/link-check')
			if (this.followRedirects) {
				uri += '?follow=true'
			}

			let self = this
			// ignore any error, do not show loader
			axios.get(uri, null)
				.then(function (result) {
					self.check = result.data
					self.error = false

					self.$emit('setLinkErrors', result.data.Errors)
				})
				.catch(function (error) {
					// handle error
					if (error.response && error.response.data) {
						// The request was made and the server responded with a status code
						// that falls out of the range of 2xx
						if (error.response.data.Error) {
							self.error = error.response.data.Error
						} else {
							self.error = error.response.data
						}
					} else if (error.request) {
						// The request was made but no response was received
						// `error.request` is an instance of XMLHttpRequest in the browser and an instance of
						// http.ClientRequest in node.js
						self.error = 'Error sending data to the server. Please try again.'
					} else {
						// Something happened in setting up the request that triggered an Error
						self.error = error.message
					}
				})
				.then(function (result) {
					// always run
					self.loading = false
				})
		},
	}
}
</script>

<template>
	<div class="pe-3">
		<div class="row mb-3 align-items-center">
			<div class="col">
				<h4 class="mb-0">
					<template v-if="!check">
						Link check
					</template>
					<template v-else>
						<template v-if="check.Links.length">
							Scanned {{ formatNumber(check.Links.length) }}
							link<template v-if="check.Links.length != 1">s</template>
						</template>
						<template v-else>
							No links detected
						</template>
					</template>
				</h4>
			</div>
			<div class="col-auto">
				<div class="input-group">
					<button class="btn btn-outline-secondary" data-bs-toggle="modal"
						data-bs-target="#AboutLinkCheckResults">
						<i class="bi bi-info-circle-fill"></i>
						Help
					</button>
					<button class="btn btn-outline-secondary" data-bs-toggle="modal" data-bs-target="#LinkCheckOptions">
						<i class="bi bi-gear-fill"></i>
						Settings
					</button>
				</div>
			</div>
		</div>

		<div v-if="!check">
			<p class="text-secondary">
				Link check scans your email text &amp; HTML for unique links, testing the response status codes.
				This includes links to images and remote CSS stylesheets.
			</p>

			<p class="text-center my-5">
				<button v-if="!check" class="btn btn-primary btn-lg" @click="doCheck()" :disabled="loading">
					<template v-if="loading">
						Checking links
						<div class="ms-1 spinner-border spinner-border-sm text-light" role="status">
							<span class="visually-hidden">Loading...</span>
						</div>
					</template>
					<template v-else>
						<i class="bi bi-check-square me-2"></i>
						Check message links
					</template>
				</button>
			</p>
		</div>

		<div v-else v-for="s, k in groupedStatuses">
			<div class="card mb-3">
				<div class="card-header h4" :class="s.Class">
					Status {{ s.StatusCode }}
					<small v-if="s.Status != ''" class="ms-2 small text-secondary">({{ s.Status }})</small>
				</div>
				<ul class="list-group list-group-flush">
					<li v-for="u in s.URLS" class="list-group-item">
						<a :href="u" target="_blank" class="no-icon">{{ u }}</a>
					</li>
				</ul>
			</div>
		</div>

		<template v-if="error">
			<p>Link check failed to load:</p>
			<div class="alert alert-warning">
				{{ error }}
			</div>
		</template>

	</div>

	<div class="modal fade" id="LinkCheckOptions" tabindex="-1" aria-labelledby="LinkCheckOptionsLabel" aria-hidden="true">
		<div class="modal-dialog modal-lg modal-dialog-scrollable">
			<div class="modal-content">
				<div class="modal-header">
					<h1 class="modal-title fs-5" id="LinkCheckOptionsLabel">Link check options</h1>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div class="modal-body">
					<h6 class="mt-4">Follow HTTP redirects (status 301 & 302)</h6>
					<div class="form-check form-switch mb-4">
						<input class="form-check-input" type="checkbox" role="switch" v-model="followRedirects"
							id="LinkCheckFollowRedirectsSwitch">
						<label class="form-check-label" for="LinkCheckFollowRedirectsSwitch">
							<template v-if="followRedirects">Following HTTP redirects</template>
							<template v-else>Not following HTTP redirects</template>
						</label>
					</div>

					<h6 class="mt-4">Automatic link checking</h6>
					<div class="form-check form-switch mb-3">
						<input class="form-check-input" type="checkbox" role="switch" v-model="autoScan"
							id="LinkCheckAutoCheckSwitch">
						<label class="form-check-label" for="LinkCheckAutoCheckSwitch">
							<template v-if="autoScan">Automatic link checking is enabled</template>
							<template v-else>Automatic link checking is disabled</template>
						</label>
						<div class="form-text">
							Note: Enabling auto checking will scan every link & image every time a message is opened.
							Only enable this if you understand the potential risks &amp; consequences.
						</div>
					</div>

				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
				</div>
			</div>
		</div>
	</div>

	<div class="modal fade" id="AboutLinkCheckResults" tabindex="-1" aria-labelledby="AboutLinkCheckResultsLabel"
		aria-hidden="true">
		<div class="modal-dialog modal-lg modal-dialog-scrollable">
			<div class="modal-content">
				<div class="modal-header">
					<h1 class="modal-title fs-5" id="AboutLinkCheckResultsLabel">About Link check</h1>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div class="modal-body">
					<div class="accordion" id="LinkCheckAboutAccordion">
						<div class="accordion-item">
							<h2 class="accordion-header">
								<button class="accordion-button collapsed" type="button" data-bs-toggle="collapse"
									data-bs-target="#col1" aria-expanded="false" aria-controls="col1">
									What is Link check?
								</button>
							</h2>
							<div id="col1" class="accordion-collapse collapse" data-bs-parent="#LinkCheckAboutAccordion">
								<div class="accordion-body">
									Link check scans your message HTML and text for all unique links, images and linked
									stylesheets. It then does a HTTP <code>HEAD</code> request to each link, 5 at a time, to
									test whether the link/image/stylesheet exists.
								</div>
							</div>
						</div>
						<div class="accordion-item">
							<h2 class="accordion-header">
								<button class="accordion-button collapsed" type="button" data-bs-toggle="collapse"
									data-bs-target="#col2" aria-expanded="false" aria-controls="col2">
									What are "301" and "302" links?
								</button>
							</h2>
							<div id="col2" class="accordion-collapse collapse" data-bs-parent="#LinkCheckAboutAccordion">
								<div class="accordion-body">
									<p>
										These are links that redirect you to another URL, for example newsletters
										often use redirect links to track user clicks.
									</p>
									<p>
										By default Link check will not follow these links, however you can turn this on via
										the settings and Link check will "follow" those redirects.
									</p>
								</div>
							</div>
						</div>
						<div class="accordion-item">
							<h2 class="accordion-header">
								<button class="accordion-button collapsed" type="button" data-bs-toggle="collapse"
									data-bs-target="#col3" aria-expanded="false" aria-controls="col3">
									Why are some links returning an error but work in my browser?
								</button>
							</h2>
							<div id="col3" class="accordion-collapse collapse" data-bs-parent="#LinkCheckAboutAccordion">
								<div class="accordion-body">
									<p>This may be due to various reasons, for instance:</p>
									<ul>
										<li>The Mailpit server cannot resolve (DNS) the hostname of the URL.</li>
										<li>Mailpit is not allowed to access the URL.</li>
										<li>
											The webserver is blocking requests that don't come from authenticated web
											browsers.
										</li>
										<li>The webserver or doesn't allow HTTP <code>HEAD</code> requests. </li>
									</ul>
								</div>
							</div>
						</div>
						<div class="accordion-item">
							<h2 class="accordion-header">
								<button class="accordion-button collapsed" type="button" data-bs-toggle="collapse"
									data-bs-target="#col4" aria-expanded="false" aria-controls="col4">
									What are the risks of running Link check automatically?
								</button>
							</h2>
							<div id="col4" class="accordion-collapse collapse" data-bs-parent="#LinkCheckAboutAccordion">
								<div class="accordion-body">
									<p>
										Depending on the type of messages you are testing, opening all links on all messages
										may have undesired consequences:
									</p>
									<ul>
										<li>If the message contains tracking links this may reveal your identity.</li>
										<li>
											If the message contains unsubscribe links, Link check could unintentionally
											unsubscribe you.
										</li>
										<li>
											To speed up the checking process, Link check will attempt 5 URLs at a time. This
											could lead to temporary heady load on the remote server.
										</li>
									</ul>
									<p>
										Unless you know what messages you receive, it is advised to only run the Link check
										manually.
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
