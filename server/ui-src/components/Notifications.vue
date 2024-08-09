<script>
import CommonMixins from '../mixins/CommonMixins'
import { Toast } from 'bootstrap'
import { mailbox } from '../stores/mailbox'
import { pagination } from '../stores/pagination'

export default {
	mixins: [CommonMixins],

	// global event bus to handle message status changes
	inject: ["eventBus"],

	data() {
		return {
			pagination,
			mailbox,
			toastMessage: false,
			reconnectRefresh: false,
			socketURI: false,
			socketLastConnection: 0, // timestamp to track reconnection times & avoid reloading mailbox on short disconnections
			socketBreaks: 0, // to track sockets that continually connect & disconnect, reset every 15s
			pauseNotifications: false, // prevent spamming
			version: false,
			paginationDelayed: false, // for delayed pagination URL changes
		}
	},

	mounted() {
		const d = document.getElementById('app')
		if (d) {
			this.version = d.dataset.version
		}

		const proto = location.protocol == 'https:' ? 'wss' : 'ws'
		this.socketURI = proto + "://" + document.location.host + this.resolve(`/api/events`)

		this.socketBreakReset()
		this.connect()

		mailbox.notificationsSupported = window.isSecureContext
			&& ("Notification" in window && Notification.permission !== "denied")
		mailbox.notificationsEnabled = mailbox.notificationsSupported && Notification.permission == "granted"
	},

	methods: {
		// websocket connect
		connect() {
			const ws = new WebSocket(this.socketURI)
			ws.onmessage = (e) => {
				let response
				try {
					response = JSON.parse(e.data)
				} catch (e) {
					return
				}

				// new messages
				if (response.Type == "new" && response.Data) {
					this.eventBus.emit("new", response.Data)

					if (!mailbox.searching) {
						if (pagination.start < 1) {
							// push results directly into first page
							mailbox.messages.unshift(response.Data)
							if (mailbox.messages.length > pagination.limit) {
								mailbox.messages.pop()
							}
						} else {
							// update pagination offset
							pagination.start++
							// prevent "Too many calls to Location or History APIs within a short time frame"
							this.delayedPaginationUpdate()
						}
					}

					for (let i in response.Data.Tags) {
						if (mailbox.tags.findIndex(e => { return e.toLowerCase() === response.Data.Tags[i].toLowerCase() }) < 0) {
							mailbox.tags.push(response.Data.Tags[i])
							mailbox.tags.sort((a, b) => {
								return a.toLowerCase().localeCompare(b.toLowerCase())
							})
						}
					}

					// send notifications
					if (!this.pauseNotifications) {
						this.pauseNotifications = true
						let from = response.Data.From != null ? response.Data.From.Address : '[unknown]'
						this.browserNotify("New mail from: " + from, response.Data.Subject)
						this.setMessageToast(response.Data)
						// delay notifications by 2s
						window.setTimeout(() => { this.pauseNotifications = false }, 2000)
					}
				} else if (response.Type == "prune") {
					// messages have been deleted, reload messages to adjust
					window.scrollInPlace = true
					mailbox.refresh = true // trigger refresh
					window.setTimeout(() => { mailbox.refresh = false }, 500)
					this.eventBus.emit("prune");
				} else if (response.Type == "stats" && response.Data) {
					// refresh mailbox stats
					mailbox.total = response.Data.Total
					mailbox.unread = response.Data.Unread

					// detect version updated, refresh is needed
					if (this.version != response.Data.Version) {
						location.reload()
					}
				} else if (response.Type == "delete" && response.Data) {
					// broadcast for components
					this.eventBus.emit("delete", response.Data)
				} else if (response.Type == "update" && response.Data) {
					// broadcast for components
					this.eventBus.emit("update", response.Data)
				} else if (response.Type == "truncate") {
					// broadcast for components
					this.eventBus.emit("truncate")
				}
			}

			ws.onopen = () => {
				mailbox.connected = true
				this.socketLastConnection = Date.now()
				if (this.reconnectRefresh) {
					this.reconnectRefresh = false
					mailbox.refresh = true // trigger refresh
					window.setTimeout(() => { mailbox.refresh = false }, 500)
				}
			}

			ws.onclose = (e) => {
				if (this.socketLastConnection == 0) {
					// connection failed immediately after connecting to Mailpit implies proxy websockets aren't configured
					console.log('Unable to connect to websocket, disabling websocket support')
					return
				}

				if (mailbox.connected) {
					// count disconnections
					this.socketBreaks++
				}

				// set disconnected state
				mailbox.connected = false

				if (this.socketBreaks > 3) {
					// give up after > 3 successful socket connections & disconnections within a 15 second window,
					// something is not working right on their end, see issue #319
					console.log('Unstable websocket connection, disabling websocket support')
					return
				}
				if (Date.now() - this.socketLastConnection > 5000) {
					// only refresh mailbox if the last successful connection was broken for > 5 seconds
					this.reconnectRefresh = true
				} else {
					this.reconnectRefresh = false
				}

				setTimeout(() => {
					this.connect() // reconnect
				}, 1000)
			}

			ws.onerror = function (err) {
				ws.close()
			}
		},

		socketBreakReset() {
			window.setTimeout(() => {
				this.socketBreaks = 0
				this.socketBreakReset()
			}, 15000)
		},

		// This will only update the pagination offset at a maximum of 2x per second
		// when viewing the inbox on > page 1, while receiving an influx of new messages.
		delayedPaginationUpdate() {
			if (this.paginationDelayed) {
				return
			}

			this.paginationDelayed = true

			window.setTimeout(() => {
				const path = this.$route.path
				const p = {
					...this.$route.query
				}
				if (pagination.start > 0) {
					p.start = pagination.start.toString()
				} else {
					delete p.start
				}
				if (pagination.limit != pagination.defaultLimit) {
					p.limit = pagination.limit.toString()
				} else {
					delete p.limit
				}

				mailbox.autoPaginating = false // prevent reload of messages when URL changes
				const params = new URLSearchParams(p)
				this.$router.replace(path + '?' + params.toString())

				this.paginationDelayed = false
			}, 500)
		},

		browserNotify(title, message) {
			if (!("Notification" in window)) {
				return
			}

			if (Notification.permission === "granted") {
				let options = {
					body: message,
					icon: this.resolve('/notification.png')
				}
				new Notification(title, options)
			}
		},

		setMessageToast(m) {
			// don't display if browser notifications are enabled, or a toast is already displayed
			if (mailbox.notificationsEnabled || this.toastMessage) {
				return
			}

			this.toastMessage = m

			const el = document.getElementById('messageToast')
			if (el) {
				el.addEventListener('hidden.bs.toast', () => {
					this.toastMessage = false
				})

				Toast.getOrCreateInstance(el).show()
			}
		},

		closeToast() {
			const el = document.getElementById('messageToast')
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
	</div>
</template>
