<script>
import CommonMixins from '../mixins/CommonMixins.js'
import AjaxLoader from './AjaxLoader.vue'

export default {
	mixins: [CommonMixins],

	components: {
		AjaxLoader
	},

	data() {
		return {
			theme: 'auto',
			icon: '#circle-half',
			icons: {
				'auto': '#circle-half',
				'light': '#sun-fill',
				'dark': '#moon-stars-fill'
			},
			appInfo: {},
		}
	},

	mounted() {
		this.setTheme(this.getPreferredTheme())
	},

	methods: {
		loadInfo: function () {
			let self = this
			self.get(this.baseURL + 'api/v1/info', false, function (response) {
				self.appInfo = response.data
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

			return window.matchMedia('(prefers-color-scheme: dark)').matches
				? 'dark'
				: 'light'
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
		}
	}
}
</script>

<template>
	<div class="position-fixed bg-body bottom-0 ms-n1 py-2 text-muted small col-lg-2 col-md-3 pe-3 z-3">
		<button class="text-muted btn btn-sm" v-on:click="loadInfo">
			<i class="bi bi-info-circle-fill"></i>
			About
		</button>

		<svg xmlns="http://www.w3.org/2000/svg" style="display: none;">
			<symbol id="bootstrap" viewBox="0 0 512 408" fill="currentcolor">
				<path
					d="M106.342 0c-29.214 0-50.827 25.58-49.86 53.32.927 26.647-.278 61.165-8.966 89.31C38.802 170.862 24.07 188.707 0 191v26c24.069 2.293 38.802 20.138 47.516 48.37 8.688 28.145 9.893 62.663 8.965 89.311C55.515 382.42 77.128 408 106.342 408h299.353c29.214 0 50.827-25.58 49.861-53.319-.928-26.648.277-61.166 8.964-89.311 8.715-28.232 23.411-46.077 47.48-48.37v-26c-24.069-2.293-38.765-20.138-47.48-48.37-8.687-28.145-9.892-62.663-8.964-89.31C456.522 25.58 434.909 0 405.695 0H106.342zm236.559 251.102c0 38.197-28.501 61.355-75.798 61.355h-87.202a2 2 0 01-2-2v-213a2 2 0 012-2h86.74c39.439 0 65.322 21.354 65.322 54.138 0 23.008-17.409 43.61-39.594 47.219v1.203c30.196 3.309 50.532 24.212 50.532 53.085zm-84.58-128.125h-45.91v64.814h38.669c29.888 0 46.373-12.03 46.373-33.535 0-20.151-14.174-31.279-39.132-31.279zm-45.91 90.53v71.431h47.605c31.12 0 47.605-12.482 47.605-35.941 0-23.46-16.947-35.49-49.608-35.49h-45.602z" />
			</symbol>
			<symbol id="check2" viewBox="0 0 16 16" fill="currentcolor">
				<path
					d="M13.854 3.646a.5.5 0 0 1 0 .708l-7 7a.5.5 0 0 1-.708 0l-3.5-3.5a.5.5 0 1 1 .708-.708L6.5 10.293l6.646-6.647a.5.5 0 0 1 .708 0z" />
			</symbol>
			<symbol id="circle-half" viewBox="0 0 16 16" fill="currentcolor">
				<path d="M8 15A7 7 0 1 0 8 1v14zm0 1A8 8 0 1 1 8 0a8 8 0 0 1 0 16z" />
			</symbol>
			<symbol id="moon-stars-fill" viewBox="0 0 16 16" fill="currentcolor">
				<path
					d="M6 .278a.768.768 0 0 1 .08.858 7.208 7.208 0 0 0-.878 3.46c0 4.021 3.278 7.277 7.318 7.277.527 0 1.04-.055 1.533-.16a.787.787 0 0 1 .81.316.733.733 0 0 1-.031.893A8.349 8.349 0 0 1 8.344 16C3.734 16 0 12.286 0 7.71 0 4.266 2.114 1.312 5.124.06A.752.752 0 0 1 6 .278z" />
				<path
					d="M10.794 3.148a.217.217 0 0 1 .412 0l.387 1.162c.173.518.579.924 1.097 1.097l1.162.387a.217.217 0 0 1 0 .412l-1.162.387a1.734 1.734 0 0 0-1.097 1.097l-.387 1.162a.217.217 0 0 1-.412 0l-.387-1.162A1.734 1.734 0 0 0 9.31 6.593l-1.162-.387a.217.217 0 0 1 0-.412l1.162-.387a1.734 1.734 0 0 0 1.097-1.097l.387-1.162zM13.863.099a.145.145 0 0 1 .274 0l.258.774c.115.346.386.617.732.732l.774.258a.145.145 0 0 1 0 .274l-.774.258a1.156 1.156 0 0 0-.732.732l-.258.774a.145.145 0 0 1-.274 0l-.258-.774a1.156 1.156 0 0 0-.732-.732l-.774-.258a.145.145 0 0 1 0-.274l.774-.258c.346-.115.617-.386.732-.732L13.863.1z" />
			</symbol>
			<symbol id="sun-fill" viewBox="0 0 16 16" fill="currentcolor">
				<path
					d="M8 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM8 0a.5.5 0 0 1 .5.5v2a.5.5 0 0 1-1 0v-2A.5.5 0 0 1 8 0zm0 13a.5.5 0 0 1 .5.5v2a.5.5 0 0 1-1 0v-2A.5.5 0 0 1 8 13zm8-5a.5.5 0 0 1-.5.5h-2a.5.5 0 0 1 0-1h2a.5.5 0 0 1 .5.5zM3 8a.5.5 0 0 1-.5.5h-2a.5.5 0 0 1 0-1h2A.5.5 0 0 1 3 8zm10.657-5.657a.5.5 0 0 1 0 .707l-1.414 1.415a.5.5 0 1 1-.707-.708l1.414-1.414a.5.5 0 0 1 .707 0zm-9.193 9.193a.5.5 0 0 1 0 .707L3.05 13.657a.5.5 0 0 1-.707-.707l1.414-1.414a.5.5 0 0 1 .707 0zm9.193 2.121a.5.5 0 0 1-.707 0l-1.414-1.414a.5.5 0 0 1 .707-.707l1.414 1.414a.5.5 0 0 1 0 .707zM4.464 4.465a.5.5 0 0 1-.707 0L2.343 3.05a.5.5 0 1 1 .707-.707l1.414 1.414a.5.5 0 0 1 0 .708z" />
			</symbol>
		</svg>
		<div class="dropdown bd-mode-toggle float-end me-2 d-inline-block">
			<button class="btn btn-sm btn-outline-secondary dropdown-toggle" type="button" aria-expanded="false"
				title="Toggle theme" data-bs-toggle="dropdown" aria-label="Toggle theme">
				<svg class="bi my-1 theme-icon-active" width="1em" height="1em">
					<use :href="icon"></use>
				</svg>
				<span class="visually-hidden" id="bd-theme-text">Toggle theme</span>
			</button>
			<ul class="dropdown-menu dropdown-menu-end shadow" aria-labelledby="bd-theme-text">
				<li>
					<button type="button" class="dropdown-item d-flex align-items-center"
						:class="theme == 'light' ? 'active' : ''" @click="setStoredTheme('light')">
						<svg class="bi me-2 opacity-50 theme-icon" width="1em" height="1em">
							<use href="#sun-fill"></use>
						</svg>
						Light
					</button>
				</li>
				<li>
					<button type="button" class="dropdown-item d-flex align-items-center"
						:class="theme == 'dark' ? 'active' : ''" @click="setStoredTheme('dark')">
						<svg class="bi me-2 opacity-50 theme-icon" width="1em" height="1em">
							<use href="#moon-stars-fill"></use>
						</svg>
						Dark
					</button>
				</li>
				<li>
					<button type="button" class="dropdown-item d-flex align-items-center"
						:class="theme == 'auto' ? 'active' : ''" @click="setStoredTheme('auto')">
						<svg class="bi me-2 opacity-50 theme-icon" width="1em" height="1em">
							<use href="#circle-half"></use>
						</svg>
						Auto
					</button>
				</li>
			</ul>
		</div>
	</div>

	<!-- Modal -->
	<div class="modal fade" id="AppInfoModal" tabindex="-1" aria-labelledby="AppInfoModalLabel" aria-hidden="true">
		<div class="modal-dialog">
			<div class="modal-content">
				<div class="modal-header" v-if="appInfo">
					<h5 class="modal-title" id="AppInfoModalLabel">
						Mailpit
						<code>({{ appInfo.Version }})</code>
					</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div class="modal-body">
					<a class="btn btn-warning d-block mb-3" v-if="appInfo.Version != appInfo.LatestVersion"
						:href="'https://github.com/axllent/mailpit/releases/tag/' + appInfo.LatestVersion">
						A new version of Mailpit ({{ appInfo.LatestVersion }}) is available.
					</a>

					<div class="row g-3">
						<div class="col-12">
							<a class="btn btn-primary w-100" href="api/v1/" target="_blank">
								<i class="bi bi-braces"></i>
								OpenAPI / Swagger API documentation
							</a>
						</div>
						<div class="col-sm-6">
							<a class="btn btn-primary w-100" href="https://github.com/axllent/mailpit" target="_blank">
								<i class="bi bi-github"></i>
								Github
							</a>
						</div>
						<div class="col-sm-6">
							<a class="btn btn-primary w-100" href="https://github.com/axllent/mailpit/wiki" target="_blank">
								Documentation
							</a>
						</div>
						<div class="col-sm-6">
							<div class="card border-secondary text-center">
								<div class="card-header">Database size</div>
								<div class="card-body text-secondary">
									<h5 class="card-title">{{ getFileSize(appInfo.DatabaseSize) }} </h5>
								</div>
							</div>
						</div>
						<div class="col-sm-6">
							<div class="card border-secondary text-center">
								<div class="card-header">RAM usage</div>
								<div class="card-body text-secondary">
									<h5 class="card-title">{{ getFileSize(appInfo.Memory) }} </h5>
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

	<AjaxLoader :loading="loading" />
</template>
