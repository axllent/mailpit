<script>
import AjaxLoader from "./AjaxLoader.vue";
import Settings from "./AppSettings.vue";
import CommonMixins from "../mixins/CommonMixins";
import { mailbox } from "../stores/mailbox";

export default {
	components: {
		AjaxLoader,
		Settings,
	},

	mixins: [CommonMixins],

	props: {
		modals: {
			type: Boolean,
			default: false,
		},
	},

	data() {
		return {
			mailbox,
		};
	},

	methods: {
		loadInfo() {
			this.get(this.resolve("/api/v1/info"), false, (response) => {
				mailbox.appInfo = response.data;
				this.modal("AppInfoModal").show();
			});
		},

		requestNotifications() {
			// check if the browser supports notifications
			if (!("Notification" in window)) {
				alert("This browser does not support desktop notifications");
			}

			// we need to ask the user for permission
			else if (Notification.permission !== "denied") {
				Notification.requestPermission().then((permission) => {
					if (permission === "granted") {
						mailbox.notificationsEnabled = true;
					}

					this.modal("EnableNotificationsModal").hide();
				});
			}
		},
	},
};
</script>

<template>
	<template v-if="!modals">
		<div class="bg-body ms-sm-n1 me-sm-n1 py-2 text-muted small about-mailpit">
			<button class="text-muted btn btn-sm" @click="loadInfo()">
				<i class="bi bi-info-circle-fill me-1"></i>
				About
			</button>

			<button
				class="btn btn-sm btn-outline-secondary float-end"
				data-bs-toggle="modal"
				data-bs-target="#SettingsModal"
				title="Mailpit UI settings"
			>
				<i class="bi bi-gear-fill"></i>
			</button>

			<button
				v-if="mailbox.connected && mailbox.notificationsSupported && !mailbox.notificationsEnabled"
				class="btn btn-sm btn-outline-secondary float-end me-2"
				data-bs-toggle="modal"
				data-bs-target="#EnableNotificationsModal"
				title="Enable browser notifications"
			>
				<i class="bi bi-bell"></i>
			</button>
		</div>
	</template>

	<template v-else>
		<!-- Modals -->
		<div
			id="AppInfoModal"
			class="modal modal-xl fade"
			tabindex="-1"
			aria-labelledby="AppInfoModalLabel"
			aria-hidden="true"
		>
			<div class="modal-dialog">
				<div v-if="mailbox.appInfo.RuntimeStats" class="modal-content">
					<div class="modal-header">
						<h5 id="AppInfoModalLabel" class="modal-title">
							Mailpit
							<code>({{ mailbox.appInfo.Version }})</code>
						</h5>
						<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
					</div>
					<div class="modal-body">
						<div class="row g-3">
							<div class="col-xl-6">
								<div v-if="mailbox.appInfo.LatestVersion != 'disabled'">
									<div v-if="mailbox.appInfo.LatestVersion == ''" class="row g-3">
										<div class="col">
											<div class="alert alert-warning mb-3">
												There might be a newer version available. The check failed.
											</div>
										</div>
									</div>
									<div
										v-else-if="mailbox.appInfo.Version != mailbox.appInfo.LatestVersion"
										class="row g-3"
									>
										<div class="col">
											<a
												class="btn btn-warning d-block mb-3"
												:href="
													'https://github.com/axllent/mailpit/releases/tag/' +
													mailbox.appInfo.LatestVersion
												"
											>
												A new version of Mailpit ({{ mailbox.appInfo.LatestVersion }}) is
												available.
											</a>
										</div>
									</div>
								</div>
								<div class="row g-3">
									<div class="col-12">
										<RouterLink to="/api/v1/" class="btn btn-primary w-100" target="_blank">
											<i class="bi bi-braces"></i>
											OpenAPI / Swagger API documentation
										</RouterLink>
									</div>
									<div class="col-sm-6">
										<a
											class="btn btn-primary w-100"
											href="https://github.com/axllent/mailpit"
											target="_blank"
										>
											<i class="bi bi-github"></i>
											Github
										</a>
									</div>
									<div class="col-sm-6">
										<a
											class="btn btn-primary w-100"
											href="https://mailpit.axllent.org/docs/"
											target="_blank"
										>
											Documentation
										</a>
									</div>
									<div class="col-6">
										<div class="card border-secondary text-center">
											<div class="card-header">Database size</div>
											<div class="card-body text-muted">
												<h5 class="card-title">
													{{ getFileSize(mailbox.appInfo.DatabaseSize) }}
												</h5>
											</div>
										</div>
									</div>
									<div class="col-6">
										<div class="card border-secondary text-center">
											<div class="card-header">RAM usage</div>
											<div class="card-body text-muted">
												<h5 class="card-title">
													{{ getFileSize(mailbox.appInfo.RuntimeStats.Memory) }}
												</h5>
											</div>
										</div>
									</div>
								</div>
							</div>
							<div class="col-xl-6">
								<div class="card border-secondary h-100">
									<div class="card-header h4">
										Runtime statistics
										<button class="btn btn-sm btn-outline-secondary float-end" @click="loadInfo()">
											Refresh
										</button>
									</div>
									<div class="card-body text-muted">
										<table class="table table-sm table-borderless mb-0">
											<tbody>
												<tr>
													<td>Mailpit up since</td>
													<td>
														{{ secondsToRelative(mailbox.appInfo.RuntimeStats.Uptime) }}
													</td>
												</tr>
												<tr>
													<td>Messages deleted</td>
													<td>
														{{ formatNumber(mailbox.appInfo.RuntimeStats.MessagesDeleted) }}
													</td>
												</tr>
												<tr>
													<td>SMTP messages accepted</td>
													<td>
														{{ formatNumber(mailbox.appInfo.RuntimeStats.SMTPAccepted) }}
														<small class="text-muted">
															({{
																getFileSize(
																	mailbox.appInfo.RuntimeStats.SMTPAcceptedSize,
																)
															}})
														</small>
													</td>
												</tr>
												<tr>
													<td>SMTP messages rejected</td>
													<td>
														{{ formatNumber(mailbox.appInfo.RuntimeStats.SMTPRejected) }}
													</td>
												</tr>
												<tr v-if="mailbox.uiConfig.DuplicatesIgnored">
													<td>SMTP messages ignored</td>
													<td>
														{{ formatNumber(mailbox.appInfo.RuntimeStats.SMTPIgnored) }}
													</td>
												</tr>
											</tbody>
										</table>
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

		<div
			id="EnableNotificationsModal"
			class="modal fade"
			tabindex="-1"
			aria-labelledby="EnableNotificationsModalLabel"
			aria-hidden="true"
		>
			<div class="modal-dialog modal-lg">
				<div class="modal-content">
					<div class="modal-header">
						<h5 id="EnableNotificationsModalLabel" class="modal-title">Enable browser notifications?</h5>
						<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
					</div>
					<div class="modal-body">
						<p class="h4">Get browser notifications when Mailpit receives new messages?</p>
						<p>
							Note that your browser will ask you for confirmation when you click
							<code>enable notifications</code>, and that you must have Mailpit open in a browser tab to
							be able to receive the notifications.
						</p>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
						<button type="button" class="btn btn-success" @click="requestNotifications">
							Enable notifications
						</button>
					</div>
				</div>
			</div>
		</div>

		<Settings />
	</template>

	<AjaxLoader :loading="loading" />
</template>
