<script>
import { mailbox } from "../stores/mailbox.js"
import { pagination } from "../stores/pagination.js"
import { Toast } from 'bootstrap'

export default {
	data() {
		return {
			pagination,
			mailbox,
			toastMessage: false, // 
			reconnectRefresh: false,
		}
	},

	mounted() {
		this.connect()

		mailbox.notificationsSupported = window.isSecureContext
			&& ("Notification" in window && Notification.permission !== "denied")
		mailbox.notificationsEnabled = mailbox.notificationsSupported && Notification.permission == "granted"
	},

	methods: {
		// websocket connect
		connect: function () {
			let proto = location.protocol == 'https:' ? 'wss' : 'ws'
			let ws = new WebSocket(
				proto + "://" + document.location.host + this.$router.resolve(`api/events`).href
			)
			let self = this
			ws.onmessage = function (e) {
				let response = JSON.parse(e.data)
				if (!response) {
					return
				}
				// new messages
				if (response.Type == "new" && response.Data) {
					if (!mailbox.searching) {
						if (pagination.start < 1) {
							// first page
							mailbox.messages.unshift(response.Data)
							if (mailbox.messages.length > pagination.limit) {
								mailbox.messages.pop()
							}
						} else {
							pagination.start++
						}
					}

					mailbox.total++
					mailbox.unread++

					for (let i in response.Data.Tags) {
						if (mailbox.tags.indexOf(response.Data.Tags[i]) < 0) {
							mailbox.tags.push(response.Data.Tags[i])
							mailbox.tags.sort()
						}
					}

					// send notifications
					let from = response.Data.From != null ? response.Data.From.Address : '[unknown]'
					self.browserNotify("New mail from: " + from, response.Data.Subject)
					self.setMessageToast(response.Data)
				} else if (response.Type == "prune") {
					// messages have been deleted, reload messages to adjust
					window.scrollInPlace = true
					mailbox.refresh = true // trigger refresh
					window.setTimeout(() => { mailbox.refresh = false }, 500)
				}
			}

			ws.onopen = function () {
				mailbox.connected = true
				if (self.reconnectRefresh) {
					self.reconnectRefresh = false
					mailbox.refresh = true // trigger refresh
					window.setTimeout(() => { mailbox.refresh = false }, 500)
				}
			}

			ws.onclose = function (e) {
				mailbox.connected = false
				self.reconnectRefresh = true

				setTimeout(function () {
					self.connect() // reconnect
				}, 1000)
			}

			ws.onerror = function (err) {
				ws.close()
			}
		},

		browserNotify: function (title, message) {
			if (!("Notification" in window)) {
				return
			}

			if (Notification.permission === "granted") {
				let b = message.Subject
				let options = {
					body: message,
					icon: 'notification.png'
				}
				new Notification(title, options)
			}
		},

		setMessageToast: function (m) {
			// don't display if browser notifications are enabled, or a toast is already displayed
			if (mailbox.notificationsEnabled || this.toastMessage) {
				return
			}

			this.toastMessage = m

			let self = this
			let el = document.getElementById('messageToast')
			if (el) {
				el.addEventListener('hidden.bs.toast', () => {
					self.toastMessage = false
				})

				Toast.getOrCreateInstance(el).show()
			}
		},

		closeToast: function () {
			let el = document.getElementById('messageToast')
			if (el) {
				Toast.getOrCreateInstance(el).hide()
			}
		},
	},
}
</script>

<template>
	<div class="toast-container position-fixed bottom-0 end-0 p-3">
		<div id="messageToast" class="toast" role="alert" aria-live="assertive" aria-atomic="true">
			<div class="toast-header" v-if="toastMessage">
				<i class="bi bi-envelope-exclamation-fill me-2"></i>
				<strong class="me-auto">
					<RouterLink :to="'/view/' + toastMessage.ID" @click="closeToast">New message</RouterLink>
				</strong>
				<button type="button" class="btn-close" data-bs-dismiss="toast" aria-label="Close"></button>
			</div>

			<div class="toast-body">
				<div>
					<RouterLink :to="'/view/' + toastMessage.ID" class="d-block text-truncate text-body-secondary"
						@click="closeToast">
						<template v-if="toastMessage.Subject != ''">{{ toastMessage.Subject }}</template>
						<template v-else>
							[ no subject ]
						</template>
					</RouterLink>
				</div>
			</div>
		</div>
	</div></template>
