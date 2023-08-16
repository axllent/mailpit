<script>
import commonMixins from './mixins.js'
import Message from './templates/Message.vue'
import MessageSummary from './templates/MessageSummary.vue'
import MessageRelease from './templates/MessageRelease.vue'
import MessageToast from './templates/MessageToast.vue'
import ThemeToggle from './templates/ThemeToggle.vue'
import moment from 'moment'
import Tinycon from 'tinycon'

export default {
	mixins: [commonMixins],

	components: {
		Message,
		MessageSummary,
		MessageRelease,
		MessageToast,
		ThemeToggle,
	},

	data() {
		return {
			currentPath: window.location.hash,
			items: [],
			limit: 50,
			total: 0,
			unread: 0,
			messagesCount: 0,
			start: 0,
			count: 0,
			tags: [],
			existingTags: [], // to pass onto components
			search: "",
			searching: false,
			isConnected: false,
			scrollInPlace: false,
			message: false,
			messagePrev: false,
			messageNext: false,
			notificationsSupported: false,
			notificationsEnabled: false,
			selected: [],
			tcStatus: 0,
			appInfo: false,
			lastLoaded: false,
			uiConfig: {},
			releaseAddresses: false,
			toastMessage: false,
		}
	},

	watch: {
		currentPath(v, old) {
			if (v && v.match(/^[a-z0-9]+-[a-z0-9]+-[a-z0-9]+-[a-z0-9]+-[a-z0-9]+$/)) {
				this.openMessage()
			} else {
				this.message = false
			}
		},
		unread(v, old) {
			if (v == this.tcStatus) {
				return
			}
			this.tcStatus = v
			if (v == 0) {
				Tinycon.reset()
			} else {
				Tinycon.setBubble(v)
			}
		}
	},

	computed: {
		canPrev: function () {
			return this.start > 0
		},
		canNext: function () {
			return this.messagesCount > (this.start + this.count)
		},
		unreadInSearch: function () {
			if (!this.searching) {
				return false
			}

			return this.items.filter(i => !i.Read).length
		}
	},

	mounted() {

		let title = document.title + ' - ' + location.hostname
		document.title = title

		this.currentPath = window.location.hash.slice(1)
		window.addEventListener('hashchange', () => {
			this.currentPath = window.location.hash.slice(1)
		})

		this.notificationsSupported = window.isSecureContext
			&& ("Notification" in window && Notification.permission !== "denied")
		this.notificationsEnabled = this.notificationsSupported && Notification.permission == "granted"

		Tinycon.setOptions({
			height: 11,
			background: '#dd0000',
			fallback: false
		})

		moment.updateLocale('en', {
			relativeTime: {
				future: "in %s",
				past: "%s ago",
				s: 'seconds',
				ss: '%d secs',
				m: "a minute",
				mm: "%d mins",
				h: "an hour",
				hh: "%d hours",
				d: "a day",
				dd: "%d days",
				w: "a week",
				ww: "%d weeks",
				M: "a month",
				MM: "%d months",
				y: "a year",
				yy: "%d years"
			}
		})

		this.connect()
		this.getUISettings()
		this.loadMessages()
	},

	methods: {
		loadMessages: function () {
			let now = Date.now()
			// prevent double loading when UI loads & websocket connects
			if (this.lastLoaded && now - this.lastLoaded < 250) {
				return
			}
			if (this.start == 0) {
				this.lastLoaded = now
			}

			let self = this
			let params = {}
			self.selected = []

			let uri = 'api/v1/messages'
			if (self.search) {
				self.searching = true
				uri = 'api/v1/search'
				params['query'] = self.search
				params['limit'] = self.limit
				if (self.start > 0) {
					params['start'] = self.start
				}
			} else {
				self.searching = false
				params['limit'] = self.limit
				if (self.start > 0) {
					params['start'] = self.start
				}
			}

			self.get(uri, params, function (response) {
				self.total = response.data.total
				self.unread = response.data.unread
				self.count = response.data.messages.length
				self.messagesCount = response.data.messages_count
				self.start = response.data.start
				self.items = response.data.messages
				self.tags = response.data.tags
				self.existingTags = JSON.parse(JSON.stringify(self.tags))

				// if pagination > 0 && results == 0 reload first page (prune)
				if (response.data.count == 0 && response.data.start > 0) {
					self.start = 0
					return self.loadMessages()
				}

				if (!self.scrollInPlace) {
					let mp = document.getElementById('message-page')
					if (mp) {
						mp.scrollTop = 0
					}
				}

				self.scrollInPlace = false
			})
		},

		getUISettings: function () {
			let self = this
			self.get('api/v1/webui', null, function (response) {
				self.uiConfig = response.data
			})
		},

		doSearch: function (e) {
			e.preventDefault()
			this.start = 0
			this.loadMessages()
		},

		tagSearch: function (e, tag) {
			e.preventDefault()
			if (tag.match(/ /)) {
				tag = '"' + tag + '"'
			}
			this.search = 'tag:' + tag
			window.location.hash = ""
			this.start = 0
			this.loadMessages()
		},

		resetSearch: function (e) {
			e.preventDefault()
			this.search = ''
			this.start = 0
			this.loadMessages()
		},

		reloadMessages: function () {
			this.search = ""
			this.start = 0
			this.loadMessages()
		},

		viewNext: function () {
			this.start = parseInt(this.start, 10) + parseInt(this.limit, 10)
			this.loadMessages()
		},

		viewPrev: function () {
			let s = this.start - this.limit
			if (s < 0) {
				s = 0
			}
			this.start = s
			this.loadMessages()
		},

		openMessage: function (id) {
			let self = this
			self.selected = []
			self.releaseAddresses = false
			self.toastMessage = false
			self.existingTags = JSON.parse(JSON.stringify(self.tags))

			let uri = 'api/v1/message/' + self.currentPath
			self.get(uri, false, function (response) {
				for (let i in self.items) {
					if (self.items[i].ID == self.currentPath) {
						if (!self.items[i].Read) {
							self.items[i].Read = true
							self.unread--
						}
					}
				}

				let d = response.data

				// replace inline images embedded as inline attachments
				if (d.HTML && d.Inline) {
					for (let i in d.Inline) {
						let a = d.Inline[i]
						if (a.ContentID != '') {
							d.HTML = d.HTML.replace(
								new RegExp('(=["\']?)(cid:' + a.ContentID + ')(["|\'|\\s|\\/|>|;])', 'g'),
								'$1' + window.location.origin + window.location.pathname + 'api/v1/message/' + d.ID + '/part/' + a.PartID + '$3'
							)
						}
						if (a.FileName.match(/^[a-zA-Z0-9\_\-\.]+$/)) {
							// some old email clients use the filename
							d.HTML = d.HTML.replace(
								new RegExp('(=["\']?)(' + a.FileName + ')(["|\'|\\s|\\/|>|;])', 'g'),
								'$1' + window.location.origin + window.location.pathname + 'api/v1/message/' + d.ID + '/part/' + a.PartID + '$3'
							)
						}
					}
				}

				// replace inline images embedded as regular attachments
				if (d.HTML && d.Attachments) {
					for (let i in d.Attachments) {
						let a = d.Attachments[i]
						if (a.ContentID != '') {
							d.HTML = d.HTML.replace(
								new RegExp('(=["\']?)(cid:' + a.ContentID + ')(["|\'|\\s|\\/|>|;])', 'g'),
								'$1' + window.location.origin + window.location.pathname + 'api/v1/message/' + d.ID + '/part/' + a.PartID + '$3'
							)
						}
						if (a.FileName.match(/^[a-zA-Z0-9\_\-\.]+$/)) {
							// some old email clients use the filename
							d.HTML = d.HTML.replace(
								new RegExp('(=["\']?)(' + a.FileName + ')(["|\'|\\s|\\/|>|;])', 'g'),
								'$1' + window.location.origin + window.location.pathname + 'api/v1/message/' + d.ID + '/part/' + a.PartID + '$3'
							)
						}
					}
				}

				self.message = d

				// generate the prev/next links based on current message list
				self.messagePrev = false
				self.messageNext = false
				let found = false

				for (let i in self.items) {
					if (self.items[i].ID == self.message.ID) {
						found = true
					} else if (found && !self.messageNext) {
						self.messageNext = self.items[i].ID
						break
					} else {
						self.messagePrev = self.items[i].ID
					}
				}
			})
		},

		// universal handler to delete current or selected messages
		deleteMessages: function () {
			let ids = []
			let self = this
			if (self.message) {
				ids.push(self.message.ID)
			} else {
				ids = JSON.parse(JSON.stringify(self.selected))
			}
			if (!ids.length) {
				return false
			}
			let uri = 'api/v1/messages'
			self.delete(uri, { 'ids': ids }, function (response) {
				window.location.hash = ""
				self.scrollInPlace = true
				self.loadMessages()
			})
		},

		// delete messages displayed in current search
		deleteSearch: function () {
			let ids = this.items.map(item => item.ID)

			if (!ids.length) {
				return false
			}

			let self = this
			let uri = 'api/v1/messages'
			self.delete(uri, { 'ids': ids }, function (response) {
				window.location.hash = ""
				self.scrollInPlace = true
				self.loadMessages()
			})
		},

		// delete all messages from mailbox
		deleteAll: function () {
			let self = this
			let uri = 'api/v1/messages'
			self.delete(uri, false, function (response) {
				window.location.hash = ""
				self.reloadMessages()
			})
		},

		// mark current message as read
		markUnread: function () {
			let self = this
			if (!self.message) {
				return false
			}
			let uri = 'api/v1/messages'
			self.put(uri, { 'read': false, 'ids': [self.message.ID] }, function (response) {
				window.location.hash = ""
				self.scrollInPlace = true
				self.loadMessages()
			})
		},

		// mark all messages in mailbox as read
		markAllRead: function () {
			let self = this
			let uri = 'api/v1/messages'
			self.put(uri, { 'read': true }, function (response) {
				window.location.hash = ""
				self.scrollInPlace = true
				self.loadMessages()
			})
		},

		// mark messages in current search as read
		markSearchRead: function () {
			let ids = this.items.map(item => item.ID)

			if (!ids.length) {
				return false
			}

			let self = this
			let uri = 'api/v1/messages'
			self.put(uri, { 'read': true, 'ids': ids }, function (response) {
				window.location.hash = ""
				self.scrollInPlace = true
				self.loadMessages()
			})
		},

		// mark selected messages as read
		markSelectedRead: function () {
			let self = this
			if (!self.selected.length) {
				return false
			}
			let uri = 'api/v1/messages'
			self.put(uri, { 'read': true, 'ids': self.selected }, function (response) {
				window.location.hash = ""
				self.scrollInPlace = true
				self.loadMessages()
			})
		},

		// mark selected messages as unread
		markSelectedUnread: function () {
			let self = this
			if (!self.selected.length) {
				return false
			}
			let uri = 'api/v1/messages'
			self.put(uri, { 'read': false, 'ids': self.selected }, function (response) {
				window.location.hash = ""
				self.scrollInPlace = true
				self.loadMessages()
			})
		},

		// test if any selected emails are unread
		selectedHasUnread: function () {
			if (!this.selected.length) {
				return false
			}
			for (let i in this.items) {
				if (this.isSelected(this.items[i].ID) && !this.items[i].Read) {
					return true
				}
			}
			return false
		},

		// test of any selected emails are read
		selectedHasRead: function () {
			if (!this.selected.length) {
				return false
			}
			for (let i in this.items) {
				if (this.isSelected(this.items[i].ID) && this.items[i].Read) {
					return true
				}
			}
			return false
		},

		// websocket connect
		connect: function () {
			let proto = location.protocol == 'https:' ? 'wss' : 'ws'
			let ws = new WebSocket(
				proto + "://" + document.location.host + document.location.pathname + "api/events"
			)
			let self = this
			ws.onmessage = function (e) {
				let response = JSON.parse(e.data)
				if (!response) {
					return
				}
				// new messages
				if (response.Type == "new" && response.Data) {
					if (!self.searching) {
						if (self.start < 1) {
							// first page
							self.items.unshift(response.Data)
							if (self.items.length > self.limit) {
								self.items.pop()
							}

							// first message was open, set messagePrev
							if (!self.messagePrev) {
								self.messagePrev = response.Data.ID
							}
						} else {
							self.start++
						}
					}
					self.total++
					self.unread++

					for (let i in response.Data.Tags) {
						if (self.tags.indexOf(response.Data.Tags[i]) < 0) {
							self.tags.push(response.Data.Tags[i])
							self.tags.sort()
						}
					}

					let from = response.Data.From != null ? response.Data.From.Address : '[unknown]'
					self.browserNotify("New mail from: " + from, response.Data.Subject)
					self.setMessageToast(response.Data)
				} else if (response.Type == "prune") {
					// messages have been deleted, reload messages to adjust
					self.scrollInPlace = true
					self.loadMessages()
				}
			}

			ws.onopen = function () {
				self.isConnected = true
				self.loadMessages()
			}

			ws.onclose = function (e) {
				self.isConnected = false

				setTimeout(function () {
					self.connect() // reconnect
				}, 1000)
			}

			ws.onerror = function (err) {
				ws.close()
			}
		},

		getPrimaryEmailTo: function (message) {
			for (let i in message.To) {
				return message.To[i].Address
			}

			return '[ Undisclosed recipients ]'
		},

		getRelativeCreated: function (message) {
			let d = new Date(message.Created)
			return moment(d).fromNow().toString()
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

		requestNotifications: function () {
			// check if the browser supports notifications
			if (!("Notification" in window)) {
				alert("This browser does not support desktop notification")
			}

			// we need to ask the user for permission
			else if (Notification.permission !== "denied") {
				let self = this
				Notification.requestPermission().then(function (permission) {
					// if the user accepts, let's create a notification
					if (permission === "granted") {
						self.browserNotify("Notifications enabled", "You will receive notifications when new mails are received.")
						self.notificationsEnabled = true
					}
				})
			}
		},

		toggleSelected: function (e, id) {
			e.preventDefault()

			if (this.isSelected(id)) {
				this.selected = this.selected.filter(function (ele) {
					return ele != id
				})
			} else {
				this.selected.push(id)
			}
		},

		selectRange: function (e, id) {
			e.preventDefault()

			let selecting = false
			let lastSelected = this.selected.length > 0 && this.selected[this.selected.length - 1]
			if (lastSelected == id) {
				this.selected = this.selected.filter(function (ele) {
					return ele != id
				})
				return
			}

			if (lastSelected === false) {
				this.selected.push(id)
				return
			}

			for (let d of this.items) {
				if (selecting) {
					if (!this.isSelected(d.ID)) {
						this.selected.push(d.ID)
					}
					if (d.ID == lastSelected || d.ID == id) {
						// reached backwards select
						break
					}
				} else if (d.ID == id || d.ID == lastSelected) {
					if (!this.isSelected(d.ID)) {
						this.selected.push(d.ID)
					}
					selecting = true
				}
			}
		},

		isSelected: function (id) {
			return this.selected.indexOf(id) != -1
		},

		inSearch: function (tag) {
			tag = tag.toLowerCase()
			if (tag.match(/ /)) {
				tag = '"' + tag + '"'
			}

			return this.search.toLowerCase().indexOf('tag:' + tag) > -1
		},

		loadInfo: function (e) {
			e.preventDefault()
			let self = this
			self.get('api/v1/info', false, function (response) {
				self.appInfo = response.data
				self.modal('AppInfoModal').show()
			})
		},

		downloadMessageBody: function (str, ext) {
			let dl = document.createElement('a')
			dl.href = "data:text/plain," + encodeURIComponent(str)
			dl.target = '_blank'
			dl.download = this.message.ID + '.' + ext
			dl.click()
		},

		initReleaseModal: function () {
			this.releaseAddresses = false
			let addresses = []
			for (let i in this.message.To) {
				addresses.push(this.message.To[i].Address)
			}
			for (let i in this.message.Cc) {
				addresses.push(this.message.Cc[i].Address)
			}
			for (let i in this.message.Bcc) {
				addresses.push(this.message.Bcc[i].Address)
			}

			// include only unique email addresses, regardless of casing
			let uAddresses = new Map(addresses.map(a => [a.toLowerCase(), a]))
			this.releaseAddresses = [...uAddresses.values()]

			let self = this
			window.setTimeout(function () {
				// delay to allow elements to load
				self.modal('ReleaseModal').show()

				window.setTimeout(function () {
					document.querySelector('#ReleaseModal input[role="combobox"]').focus()
				}, 500)
			}, 300)
		},

		setMessageToast: function (m) {
			// don't display if browser notifications are enabled, or a toast is already displayed
			if (this.notificationsEnabled || this.toastMessage) {
				return
			}

			this.toastMessage = m
		},

		clearMessageToast: function () {
			this.toastMessage = false
		}
	}
}
</script>

<template>
	<div class="navbar navbar-expand-lg navbar-dark row flex-shrink-0 bg-primary text-white">
		<div class="col-lg-2 col-md-3 d-none d-md-block">
			<a class="navbar-brand text-white" href="#" v-on:click="reloadMessages">
				<img src="mailpit.svg" alt="Mailpit">
				<span class="ms-2">Mailpit</span>
			</a>
		</div>

		<div class="col col-md-9 col-lg-10" v-if="message">
			<a class="btn btn-outline-light me-4 px-3 d-md-none" href="#" v-on:click="message = false"
				title="Return to messages">
				<i class="bi bi-arrow-return-left"></i>
			</a>
			<button class="btn btn-outline-light me-2" title="Mark unread" v-on:click="markUnread">
				<i class="bi bi-eye-slash"></i> <span class="d-none d-md-inline">Mark unread</span>
			</button>
			<button class="btn btn-outline-light me-2" title="Release message"
				v-if="uiConfig.MessageRelay && uiConfig.MessageRelay.Enabled" v-on:click="initReleaseModal">
				<i class="bi bi-send"></i> <span class="d-none d-md-inline">Release</span>
			</button>
			<button class="btn btn-outline-light me-2" title="Delete message" v-on:click="deleteMessages">
				<i class="bi bi-trash-fill"></i> <span class="d-none d-md-inline">Delete</span>
			</button>
			<a class="btn btn-outline-light float-end" :class="messageNext ? '' : 'disabled'" :href="'#' + messageNext"
				title="View next message">
				<i class="bi bi-caret-right-fill"></i>
			</a>
			<a class="btn btn-outline-light ms-2 me-1 float-end" :class="messagePrev ? '' : 'disabled'"
				:href="'#' + messagePrev" title="View previous message">
				<i class="bi bi-caret-left-fill"></i>
			</a>
			<div class="dropdown float-end" id="DownloadBtn">
				<button type="button" class="btn btn-outline-light dropdown-toggle" data-bs-toggle="dropdown"
					aria-expanded="false">
					<i class="bi bi-file-arrow-down-fill"></i>
					<span class="d-none d-md-inline ms-1">Download</span>
				</button>
				<ul class="dropdown-menu dropdown-menu-end">
					<li>
						<a :href="'api/v1/message/' + message.ID + '/raw?dl=1'" class="dropdown-item"
							title="Message source including headers, body and attachments">
							Raw message
						</a>
					</li>
					<li v-if="message.HTML">
						<button v-on:click="downloadMessageBody(message.HTML, 'html')" class="dropdown-item">
							HTML body
						</button>
					</li>
					<li v-if="message.Text">
						<button v-on:click="downloadMessageBody(message.Text, 'txt')" class="dropdown-item">
							Text body
						</button>
					</li>
					<li v-if="allAttachments(message).length">
						<hr class="dropdown-divider">
					</li>
					<li v-for="part in allAttachments(message)">
						<a :href="'api/v1/message/' + message.ID + '/part/' + part.PartID" type="button"
							class="row m-0 dropdown-item d-flex" target="_blank"
							:title="part.FileName != '' ? part.FileName : '[ unknown ]'" style="min-width: 350px">
							<div class="col-auto p-0 pe-1">
								<i class="bi" :class="attachmentIcon(part)"></i>
							</div>
							<div class="col text-truncate p-0 pe-1">
								{{ part.FileName != '' ? part.FileName : '[ unknown ]' }}
							</div>
							<div class="col-auto text-muted small p-0">
								{{ getFileSize(part.Size) }}
							</div>
						</a>
					</li>
				</ul>
			</div>
		</div>

		<div class="col col-md-9 col-lg-5" v-if="!message">
			<form v-on:submit="doSearch">
				<div class="input-group">
					<a class="navbar-brand d-md-none" href="#" v-on:click="reloadMessages">
						<img src="mailpit.svg" alt="Mailpit">
						<span v-if="!total" class="ms-2">Mailpit</span>
					</a>
					<div v-if="total" class="ms-md-2 d-flex border bg-body rounded-start flex-fill position-relative">
						<input type="text" class="form-control border-0" aria-label="Search" v-model.trim="search"
							placeholder="Search mailbox">
						<span class="btn btn-link position-absolute end-0 text-muted" v-if="search"
							v-on:click="resetSearch"><i class="bi bi-x-circle"></i></span>
					</div>
					<button v-if="total" class="btn btn-outline-secondary" type="submit">
						<i class="bi bi-search"></i>
					</button>
				</div>
			</form>
		</div>
		<div class="col-12 col-lg-5 text-end mt-2 mt-lg-0" v-if="!message && total">
			<button v-if="searching && items.length" class="btn btn-danger float-start d-md-none me-2"
				data-bs-toggle="modal" data-bs-target="#DeleteSearchModal" :disabled="!items" title="Delete results">
				<i class="bi bi-trash-fill"></i>
			</button>
			<button v-else class="btn btn-danger float-start d-md-none me-2" data-bs-toggle="modal"
				data-bs-target="#DeleteAllModal" :disabled="!total || searching" title="Delete all messages">
				<i class="bi bi-trash-fill"></i>
			</button>

			<button v-if="searching && items.length" class="btn btn-light float-start d-md-none" data-bs-toggle="modal"
				data-bs-target="#MarkSearchReadModal" :disabled="!unreadInSearch"
				:title="'Mark ' + formatNumber(unreadInSearch) + ' read'">
				<i class="bi bi-check2-square"></i>
			</button>
			<button v-else class="btn btn-light float-start d-md-none" data-bs-toggle="modal"
				data-bs-target="#MarkAllReadModal" :disabled="!unread || searching">
				<i class="bi bi-check2-square"></i>
			</button>

			<select v-model="limit" v-on:change="loadMessages"
				class="form-select form-select-sm d-none d-md-inline w-auto me-2">
				<option value="25">25</option>
				<option value="50">50</option>
				<option value="100">100</option>
				<option value="200">200</option>
			</select>

			<small>
				{{ formatNumber(start + 1) }}-{{ formatNumber(start + items.length) }} <small>of</small>
				{{ formatNumber(messagesCount) }}
			</small>
			<button class="btn btn-outline-light ms-2 me-1" :disabled="!canPrev" v-on:click="viewPrev"
				:title="'View previous ' + limit + ' messages'">
				<i class="bi bi-caret-left-fill"></i>
			</button>
			<button class="btn btn-outline-light" :disabled="!canNext" v-on:click="viewNext"
				:title="'View next ' + limit + ' messages'">
				<i class="bi bi-caret-right-fill"></i>
			</button>
		</div>
	</div>
	<div class="row flex-fill" style="min-height:0">
		<div class="d-none d-md-block col-lg-2 col-md-3 mh-100 position-relative"
			style="overflow-y: auto; overflow-x: hidden;">

			<div class="list-group my-2">
				<a href="#" v-on:click="message ? message = false : reloadMessages()"
					class="list-group-item list-group-item-action" :class="!searching && !message ? 'active' : ''">
					<template v-if="isConnected">
						<i class="bi bi-envelope-fill me-1" v-if="!searching && !message"></i>
						<i class="bi bi-arrow-return-left me-1" v-else></i>
					</template>
					<i class="bi bi-arrow-clockwise me-1" v-else></i>
					<span v-if="message" class="ms-1 me-1">Return</span>
					<span v-else class="ms-1">Inbox</span>
					<span class="badge rounded-pill ms-1 float-end text-bg-secondary" title="Unread messages">
						{{ formatNumber(unread) }}
					</span>
				</a>

				<template v-if="!message && !selected.length">
					<button v-if="searching && items.length" class="list-group-item list-group-item-action"
						data-bs-toggle="modal" data-bs-target="#MarkSearchReadModal" :disabled="!unreadInSearch">
						<i class="bi bi-eye-fill me-1"></i>
						Mark {{ formatNumber(unreadInSearch) }} read
					</button>
					<button v-else class="list-group-item list-group-item-action" data-bs-toggle="modal"
						data-bs-target="#MarkAllReadModal" :disabled="!unread || searching">
						<i class="bi bi-eye-fill me-1"></i>
						Mark all read
					</button>

					<button v-if="searching && items.length" class="list-group-item list-group-item-action"
						data-bs-toggle="modal" data-bs-target="#DeleteSearchModal" :disabled="!items">
						<i class="bi bi-trash-fill me-1 text-danger"></i>
						Delete {{ formatNumber(items.length) }} message<span v-if="items.length > 1">s</span>
					</button>
					<button v-else class="list-group-item list-group-item-action" data-bs-toggle="modal"
						data-bs-target="#DeleteAllModal" :disabled="!total || searching">
						<i class="bi bi-trash-fill me-1 text-danger"></i>
						Delete all
					</button>

					<button class="list-group-item list-group-item-action" data-bs-toggle="modal"
						data-bs-target="#EnableNotificationsModal"
						v-if="isConnected && notificationsSupported && !notificationsEnabled">
						<i class="bi bi-bell me-1"></i>
						Enable alerts
					</button>
				</template>
				<template v-if="!message && selected.length">
					<button class="list-group-item list-group-item-action" :disabled="!selectedHasUnread()"
						v-on:click="markSelectedRead">
						<i class="bi bi-eye-fill me-1"></i>
						Mark read
					</button>
					<button class="list-group-item list-group-item-action" :disabled="!selectedHasRead()"
						v-on:click="markSelectedUnread">
						<i class="bi bi-eye-slash me-1"></i>
						Mark unread
					</button>
					<button class="list-group-item list-group-item-action" v-on:click="deleteMessages">
						<i class="bi bi-trash-fill me-1 text-danger"></i>
						Delete selected
					</button>
					<button class="list-group-item list-group-item-action" v-on:click="selected = []">
						<i class="bi bi-x-circle me-1"></i>
						Cancel selection
					</button>
				</template>
			</div>

			<template v-if="!selected.length && tags.length && !message">
				<div class="mt-4 text-muted">
					<button class="btn btn-sm dropdown-toggle ms-n1" data-bs-toggle="dropdown" aria-expanded="false">
						Tags
					</button>
					<ul class="dropdown-menu dropdown-menu-end">
						<li>
							<button class="dropdown-item" @click="toggleTagColors()">
								<template v-if="showTagColors">Hide</template>
								<template v-else>Show</template>
								tag colors
							</button>
						</li>
					</ul>
				</div>
				<div class="list-group mt-1 mb-5 pb-3">
					<button class="list-group-item list-group-item-action small px-2" v-for="tag in tags"
						:style="showTagColors ? { borderLeftColor: colorHash(tag), borderLeftWidth: '4px' } : ''"
						v-on:click="tagSearch($event, tag)" :class="inSearch(tag) ? 'active' : ''">
						<i class="bi bi-tag-fill" v-if="inSearch(tag)"></i>
						<i class="bi bi-tag" v-else></i>
						{{ tag }}
					</button>
				</div>
			</template>

			<MessageSummary v-if="message" :message="message"></MessageSummary>

			<div class="position-fixed bg-body bottom-0 ms-n1 py-2 text-muted small col-lg-2 col-md-3 pe-3 z-3">
				<a href="#" class="text-muted btn btn-sm" v-on:click="loadInfo">
					<i class="bi bi-info-circle-fill"></i>
					About
				</a>

				<ThemeToggle />
			</div>
		</div>

		<div class="col-lg-10 col-md-9 mh-100 ps-0 ps-md-2 pe-0">
			<div class="mh-100" style="overflow-y: auto;" :class="message ? 'd-none' : ''" id="message-page">
				<div class="list-group my-2" v-if="items.length">
					<a v-for="message in items" :href="'#' + message.ID" :key="message.ID"
						v-on:click.ctrl="toggleSelected($event, message.ID)"
						v-on:click.shift="selectRange($event, message.ID)"
						class="row gx-1 message d-flex small list-group-item list-group-item-action border-start-0 border-end-0"
						:class="message.Read ? 'read' : '', isSelected(message.ID) ? 'selected' : ''">
						<div class="col-lg-3">
							<div class="d-lg-none float-end text-muted text-nowrap small">
								<i class="bi bi-paperclip h6 me-1" v-if="message.Attachments"></i>
								{{ getRelativeCreated(message) }}
							</div>
							<div class="text-truncate d-lg-none privacy">
								<span v-if="message.From" :title="message.From.Address">{{
									message.From.Name ?
									message.From.Name : message.From.Address
								}}</span>
							</div>
							<div class="text-truncate d-none d-lg-block privacy">
								<b v-if="message.From" :title="message.From.Address">{{
									message.From.Name ?
									message.From.Name : message.From.Address
								}}</b>
							</div>
							<div class="d-none d-lg-block text-truncate text-muted small privacy">
								{{ getPrimaryEmailTo(message) }}
								<span v-if="message.To && message.To.length > 1">
									[+{{ message.To.length - 1 }}]
								</span>
							</div>
						</div>
						<div class="col-lg-6 col-xxl-7 mt-2 mt-lg-0">
							<div><b>{{ message.Subject != "" ? message.Subject : "[ no subject ]" }}</b></div>
							<div>
								<span class="badge me-1" v-for="t in message.Tags"
									:style="showTagColors ? { backgroundColor: colorHash(t) } : { backgroundColor: '#6c757d' }"
									:title="'Filter messages tagged with ' + t" v-on:click="tagSearch($event, t)">
									{{ t }}
								</span>
							</div>
						</div>
						<div class="d-none d-lg-block col-1 small text-end text-muted">
							<i class="bi bi-paperclip float-start h6" v-if="message.Attachments"></i>
							{{ getFileSize(message.Size) }}
						</div>
						<div class="d-none d-lg-block col-2 col-xxl-1 small text-end text-muted">
							{{ getRelativeCreated(message) }}
						</div>
					</a>
				</div>
				<div v-else class="text-muted my-3">
					<span v-if="searching">
						No results matching your search
					</span>
					<span v-else>
						There are no emails in your mailbox
					</span>
				</div>
			</div>

			<Message v-if="message" :message="message" :existingTags="existingTags" :uiConfig="uiConfig" :key="message.ID"
				@load-messages="loadMessages">
			</Message>
		</div>
		<div id="loading" v-if="loading">
			<div class="d-flex justify-content-center align-items-center h-100">
				<div class="spinner-border text-secondary" role="status">
					<span class="visually-hidden">Loading...</span>
				</div>
			</div>
		</div>
	</div>

	<!-- Modal -->
	<div class="modal fade" id="DeleteAllModal" tabindex="-1" aria-labelledby="DeleteAllModalLabel" aria-hidden="true">
		<div class="modal-dialog">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title" id="DeleteAllModalLabel">Delete all messages?</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div class="modal-body">
					This will permanently delete {{ formatNumber(total) }} message<span v-if="total > 1">s</span>.
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
					<button type="button" class="btn btn-danger" data-bs-dismiss="modal"
						v-on:click="deleteAll">Delete</button>
				</div>
			</div>
		</div>
	</div>

	<!-- Modal -->
	<div class="modal fade" id="DeleteSearchModal" tabindex="-1" aria-labelledby="DeleteSearchModalLabel"
		aria-hidden="true">
		<div class="modal-dialog">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title" id="DeleteSearchModalLabel">
						Delete {{ formatNumber(items.length) }} search result<span v-if="items.length > 1">s</span>?
					</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div class="modal-body">
					This will permanently delete {{ formatNumber(items.length) }} message<span v-if="total > 1">s</span>.
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
					<button type="button" class="btn btn-danger" data-bs-dismiss="modal"
						v-on:click="deleteSearch">Delete</button>
				</div>
			</div>
		</div>
	</div>

	<!-- Modal -->
	<div class="modal fade" id="MarkAllReadModal" tabindex="-1" aria-labelledby="MarkAllReadModalLabel" aria-hidden="true">
		<div class="modal-dialog">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title" id="MarkAllReadModalLabel">Mark all messages as read?</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div class="modal-body">
					This will mark {{ formatNumber(unread) }} message<span v-if="unread > 1">s</span> as read.
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
					<button type="button" class="btn btn-success" data-bs-dismiss="modal"
						v-on:click="markAllRead">Confirm</button>
				</div>
			</div>
		</div>
	</div>

	<!-- Modal -->
	<div class="modal fade" id="MarkSearchReadModal" tabindex="-1" aria-labelledby="MarkSearchReadModalLabel"
		aria-hidden="true">
		<div class="modal-dialog">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title" id="MarkSearchReadModalLabel">
						Mark {{ formatNumber(unreadInSearch) }} search result<span v-if="unreadInSearch > 1">s</span> as
						read?
					</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div class="modal-body">
					This will mark {{ formatNumber(unreadInSearch) }} message<span v-if="unread > 1">s</span> as read.
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
					<button type="button" class="btn btn-success" data-bs-dismiss="modal"
						v-on:click="markSearchRead">Confirm</button>
				</div>
			</div>
		</div>
	</div>

	<!-- Modal -->
	<div class="modal fade" id="EnableNotificationsModal" tabindex="-1" aria-labelledby="EnableNotificationsModalLabel"
		aria-hidden="true">
		<div class="modal-dialog modal-lg">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title" id="EnableNotificationsModalLabel">Enable browser notifications?</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div class="modal-body">
					<p class="h4">Get browser notifications when Mailpit receives a new mail?</p>
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

	<!-- Modal -->
	<div class="modal fade" id="ReleaseModal" tabindex="-1" aria-labelledby="AppInfoModalLabel" aria-hidden="true">
		<MessageRelease v-if="releaseAddresses" :message="message" :uiConfig="uiConfig"
			:releaseAddresses="releaseAddresses"></MessageRelease>
	</div>

	<MessageToast v-if="toastMessage" :message="toastMessage" @clearMessageToast="clearMessageToast"></MessageToast>
</template>
