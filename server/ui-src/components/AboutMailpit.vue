<script>
import AjaxLoader from './AjaxLoader.vue'
import CommonMixins from '../mixins/CommonMixins'
import { mailbox } from '../stores/mailbox'

export default {
	mixins: [CommonMixins],

	components: {
		AjaxLoader
	},

	props: {
		modals: {
			type: Boolean,
			default: false,
		}
	},

	data() {
		return {
			mailbox,
			theme: 'auto',
			icon: 'circle-half',
			icons: {
				'auto': 'circle-half',
				'light': 'sun-fill',
				'dark': 'moon-stars-fill'
			},
		}
	},

	mounted() {
		this.setTheme(this.getPreferredTheme())
	},

	methods: {
		loadInfo: function () {
			let self = this
			self.get(self.resolve('/api/v1/info'), false, function (response) {
				mailbox.appInfo = response.data
				self.modal('AppInfoModal').show()
			})
		},

		getStoredTheme: function () {
			let theme = localStorage.getItem('theme')
			if (!theme) {
				theme = 'auto'
			}

			return theme
		},

		setStoredTheme: function (theme) {
			localStorage.setItem('theme', theme)
			this.setTheme(theme)
		},

		getPreferredTheme: function () {
			const storedTheme = this.getStoredTheme()
			if (storedTheme) {
				return storedTheme
			}

			return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
		},

		setTheme: function (theme) {
			this.icon = this.icons[theme]
			this.theme = theme
			if (
				theme === 'auto' &&
				window.matchMedia('(prefers-color-scheme: dark)').matches
			) {
				document.documentElement.setAttribute('data-bs-theme', 'dark')
			} else {
				document.documentElement.setAttribute('data-bs-theme', theme)
			}
		},

		requestNotifications: function () {
			// check if the browser supports notifications
			if (!("Notification" in window)) {
				alert("This browser does not support desktop notification")
			}

			// we need to ask the user for permission
			else if (Notification.permission !== "denied") {
				let self = this
				Notification.requestPermission().then(function (permission) {
					if (permission === "granted") {
						mailbox.notificationsEnabled = true
					}
				})
			}
		},
	}
}
</script>

<template>
	<template v-if="!modals">
		<div class="position-fixed bg-body bottom-0 ms-n1 py-2 text-muted small col-xl-2 col-md-3 pe-3 z-3 about-mailpit">
			<button class="text-muted btn btn-sm" v-on:click="loadInfo">
				<i class="bi bi-info-circle-fill me-1"></i>
				About
			</button>

			<div class="dropdown bd-mode-toggle float-end me-2 d-inline-block">
				<button class="btn btn-sm btn-outline-secondary dropdown-toggle" type="button" aria-expanded="false"
					title="Toggle theme" data-bs-toggle="dropdown" aria-label="Toggle theme">
					<i :class="'bi bi-' + icon + ' my-1'"></i>
					<span class="visually-hidden" id="bd-theme-text">Toggle theme</span>
				</button>
				<ul class="dropdown-menu dropdown-menu-end shadow" aria-labelledby="bd-theme-text">
					<li>
						<button type="button" class="dropdown-item d-flex align-items-center"
							:class="theme == 'light' ? 'active' : ''" @click="setStoredTheme('light')">
							<i class="bi bi-sun-fill me-2 opacity-50"></i>
							Light
						</button>
					</li>
					<li>
						<button type="button" class="dropdown-item d-flex align-items-center"
							:class="theme == 'dark' ? 'active' : ''" @click="setStoredTheme('dark')">
							<i class="bi bi-moon-stars-fill me-2 opacity-50"></i>
							Dark
						</button>
					</li>
					<li>
						<button type="button" class="dropdown-item d-flex align-items-center"
							:class="theme == 'auto' ? 'active' : ''" @click="setStoredTheme('auto')">
							<i class="bi bi-circle-half me-2 opacity-50"></i>
							Auto
						</button>
					</li>
				</ul>
			</div>

			<button class="btn btn-sm btn-outline-secondary float-end me-2" data-bs-toggle="modal"
				data-bs-target="#EnableNotificationsModal" title="Enable browser notifications"
				v-if="mailbox.connected && mailbox.notificationsSupported && !mailbox.notificationsEnabled">
				<i class="bi bi-bell"></i>
			</button>
		</div>
	</template>

	<template v-else>
		<!-- Modals -->
		<div class="modal fade" id="AppInfoModal" tabindex="-1" aria-labelledby="AppInfoModalLabel" aria-hidden="true">
			<div class="modal-dialog">
				<div class="modal-content">
					<div class="modal-header" v-if="mailbox.appInfo">
						<h5 class="modal-title" id="AppInfoModalLabel">
							Mailpit
							<code>({{ mailbox.appInfo.Version }})</code>
						</h5>
						<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
					</div>
					<div class="modal-body">
						<a class="btn btn-warning d-block mb-3"
							v-if="mailbox.appInfo.Version != mailbox.appInfo.LatestVersion"
							:href="'https://github.com/axllent/mailpit/releases/tag/' + mailbox.appInfo.LatestVersion">
							A new version of Mailpit ({{ mailbox.appInfo.LatestVersion }}) is available.
						</a>

						<div class="row g-3">
							<div class="col-12">
								<RouterLink to="/api/v1/" class="btn btn-primary w-100" target="_blank">
									<i class="bi bi-braces"></i>
									OpenAPI / Swagger API documentation
								</RouterLink>
							</div>
							<div class="col-sm-6">
								<a class="btn btn-primary w-100" href="https://github.com/axllent/mailpit" target="_blank">
									<i class="bi bi-github"></i>
									Github
								</a>
							</div>
							<div class="col-sm-6">
								<a class="btn btn-primary w-100" href="https://github.com/axllent/mailpit/wiki"
									target="_blank">
									Documentation
								</a>
							</div>
							<div class="col-6">
								<div class="card border-secondary text-center">
									<div class="card-header">Database size</div>
									<div class="card-body text-secondary">
										<h5 class="card-title">{{ getFileSize(mailbox.appInfo.DatabaseSize) }} </h5>
									</div>
								</div>
							</div>
							<div class="col-6">
								<div class="card border-secondary text-center">
									<div class="card-header">RAM usage</div>
									<div class="card-body text-secondary">
										<h5 class="card-title">{{ getFileSize(mailbox.appInfo.Memory) }} </h5>
									</div>
								</div>
							</div>
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Close</button>
					</div>
				</div>
			</div>
		</div>

		<div class="modal fade" id="EnableNotificationsModal" tabindex="-1" aria-labelledby="EnableNotificationsModalLabel"
			aria-hidden="true">
			<div class="modal-dialog modal-lg">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title" id="EnableNotificationsModalLabel">Enable browser notifications?</h5>
						<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
					</div>
					<div class="modal-body">
						<p class="h4">Get browser notifications when Mailpit receives new messages?</p>
						<p>
							Note that your browser will ask you for confirmation when you click
							<code>enable notifications</code>,
							and that you must have Mailpit open in a browser tab to be able to receive the notifications.
						</p>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
						<button type="button" class="btn btn-success" data-bs-dismiss="modal"
							v-on:click="requestNotifications">Enable notifications</button>
					</div>
				</div>
			</div>
		</div>
	</template>

	<AjaxLoader :loading="loading" />
</template>
