<script>
import commonMixins from './mixins.js';
import Message from './templates/Message.vue';
import moment from 'moment';
import Tinycon from 'tinycon';

export default {
	mixins: [commonMixins],

	components: {
		Message
	},

	data() {
		return {
			currentPath: window.location.hash,
			items: [],
			limit: 50,
			total: 0,
			unread: 0,
			start: 0,
			count: 0,
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
			lastLoaded: false
		}
	},

	watch: {
		currentPath(v, old) {
			if (v && v.match(/^[a-z0-9]+-[a-z0-9]+-[a-z0-9]+-[a-z0-9]+-[a-z0-9]+$/)) {
				this.openMessage();
			} else {
				this.message = false;
			}
		},
		unread(v, old) {
			if (v == this.tcStatus) {
				return;
			}
			this.tcStatus = v;
			if (v == 0) {
				Tinycon.reset();
			} else {
				Tinycon.setBubble(v);
			}
		}
	},

	computed: {
		canPrev: function () {
			return this.start > 0;
		},
		canNext: function () {
			return this.total > (this.start + this.count);
		}
	},

	mounted() {
		this.currentPath = window.location.hash.slice(1);
		window.addEventListener('hashchange', () => {
			this.currentPath = window.location.hash.slice(1);
		});

		this.notificationsSupported = 'https:' == document.location.protocol
			&& ("Notification" in window && Notification.permission !== "denied");
		this.notificationsEnabled = this.notificationsSupported && Notification.permission == "granted";

		Tinycon.setOptions({
			height: 11,
			background: '#dd0000',
			fallback: false
		});

		this.connect();
		this.loadMessages();
	},

	methods: {
		loadMessages: function () {
			let now = Date.now()
			// prevent double loading when UI loads & websocket connects
			if (this.lastLoaded && now - this.lastLoaded < 250) {
				return;
			}
			if (this.start == 0) {
				this.lastLoaded = now;
			}

			let self = this;
			let params = {};
			this.selected = [];

			let uri = 'api/v1/messages';
			if (self.search) {
				self.searching = true;
				self.items = [];
				uri = 'api/v1/search'
				self.start = 0; // search is displayed on one page
				params['query'] = self.search;
				params['limit'] = 200;
			} else {
				self.searching = false;
				params['limit'] = self.limit;
				if (self.start > 0) {
					params['start'] = self.start;
				}
			}

			self.get(uri, params, function (response) {
				self.total = response.data.total;
				self.unread = response.data.unread;
				self.count = response.data.count;
				self.start = response.data.start;
				self.items = response.data.messages;

				// if pagination > 0 && results == 0 reload first page (prune)
				if (response.data.count == 0 && response.data.start > 0) {
					self.start = 0;
					return self.loadMessages();
				}

				if (!self.scrollInPlace) {
					let mp = document.getElementById('message-page');
					if (mp) {
						mp.scrollTop = 0;
					}
				}

				self.scrollInPlace = false;
			});
		},

		doSearch: function (e) {
			e.preventDefault();
			this.loadMessages();
		},

		resetSearch: function (e) {
			e.preventDefault();
			this.search = '';
			this.scrollInPlace = true;
			this.loadMessages();
		},

		reloadMessages: function () {
			this.search = "";
			this.start = 0;
			this.loadMessages();
		},

		viewNext: function () {
			this.start = parseInt(this.start, 10) + parseInt(this.limit, 10);
			this.loadMessages();
		},

		viewPrev: function () {
			let s = this.start - this.limit;
			if (s < 0) {
				s = 0;
			}
			this.start = s;
			this.loadMessages();
		},

		openMessage: function (id) {
			let self = this;
			self.selected = [];

			let uri = 'api/v1/message/' + self.currentPath
			self.get(uri, false, function (response) {
				for (let i in self.items) {
					if (self.items[i].ID == self.currentPath) {
						if (!self.items[i].Read) {
							self.items[i].Read = true;
							self.unread--;
						}
					}
				}
				let d = response.data;
				// replace inline images
				if (d.HTML && d.Inline) {
					for (let i in d.Inline) {
						let a = d.Inline[i];
						if (a.ContentID != '') {
							d.HTML = d.HTML.replace(
								new RegExp('cid:' + a.ContentID, 'g'),
								window.location.origin + window.location.pathname + 'api/v1/message/' + d.ID + '/part/' + a.PartID
							);
						}
						if (a.FileName.match(/^[a-zA-Z0-9\_\-\.]+$/)) {
							// some old email clients use the filename
							d.HTML = d.HTML.replace(
								new RegExp('src=(\'|")' + a.FileName + '(\'|")', 'g'),
								'src="' + window.location.origin + window.location.pathname + 'api/v1/message/' + d.ID + '/part/' + a.PartID + '"'
							);
						}
					}
				}
				// replace inline images
				if (d.HTML && d.Attachments) {
					for (let i in d.Attachments) {
						let a = d.Attachments[i];
						if (a.ContentID != '') {
							d.HTML = d.HTML.replace(
								new RegExp('cid:' + a.ContentID, 'g'),
								window.location.origin + window.location.pathname + 'api/v1/message/' + d.ID + '/part/' + a.PartID
							);
						}
						if (a.FileName.match(/^[a-zA-Z0-9\_\-\.]+$/)) {
							// some old email clients use the filename
							d.HTML = d.HTML.replace(
								new RegExp('src=(\'|")' + a.FileName + '(\'|")', 'g'),
								'src="' + window.location.origin + window.location.pathname + 'api/v1/message/' + d.ID + '/part/' + a.PartID + '"'
							);
						}
					}
				}

				self.message = d;
				// generate the prev/next links based on current message list
				self.messagePrev = false;
				self.messageNext = false;
				let found = false;
				for (let i in self.items) {
					if (self.items[i].ID == self.message.ID) {
						found = true;
					} else if (found && !self.messageNext) {
						self.messageNext = self.items[i].ID;
						break;
					} else {
						self.messagePrev = self.items[i].ID;
					}
				}
			});
		},

		// universal handler to delete current or selected messages
		deleteMessages: function () {
			let ids = [];
			let self = this;
			if (self.message) {
				ids.push(self.message.ID);
			} else {
				ids = JSON.parse(JSON.stringify(self.selected));
			}
			if (!ids.length) {
				return false;
			}
			let uri = 'api/v1/messages';
			self.delete(uri, { 'ids': ids }, function (response) {
				window.location.hash = "";
				self.scrollInPlace = true;
				self.loadMessages();
			});
		},

		deleteAll: function () {
			let self = this;
			let uri = 'api/v1/messages';
			self.delete(uri, false, function (response) {
				window.location.hash = "";
				self.reloadMessages();
			});
		},

		markUnread: function () {
			let self = this;
			if (!self.message) {
				return false;
			}
			let uri = 'api/v1/messages';
			self.put(uri, { 'read': false, 'ids': [self.message.ID] }, function (response) {
				window.location.hash = "";
				self.scrollInPlace = true;
				self.loadMessages();
			});
		},

		markAllRead: function () {
			let self = this;
			let uri = 'api/v1/messages'
			self.put(uri, { 'read': true }, function (response) {
				window.location.hash = "";
				self.scrollInPlace = true;
				self.loadMessages();
			});
		},

		markSelectedRead: function () {
			let self = this;
			if (!self.selected.length) {
				return false;
			}
			let uri = 'api/v1/messages';
			self.put(uri, { 'read': true, 'ids': self.selected }, function (response) {
				window.location.hash = "";
				self.scrollInPlace = true;
				self.loadMessages();
			});
		},

		markSelectedUnread: function () {
			let self = this;
			if (!self.selected.length) {
				return false;
			}
			let uri = 'api/v1/messages';
			self.put(uri, { 'read': false, 'ids': self.selected }, function (response) {
				window.location.hash = "";
				self.scrollInPlace = true;
				self.loadMessages();
			});
		},

		// test of any selected emails are unread
		selectedHasUnread: function () {
			if (!this.selected.length) {
				return false;
			}
			for (let i in this.items) {
				if (this.isSelected(this.items[i].ID) && !this.items[i].Read) {
					return true;
				}
			}
			return false;
		},

		// test of any selected emails are read
		selectedHasRead: function () {
			if (!this.selected.length) {
				return false;
			}
			for (let i in this.items) {
				if (this.isSelected(this.items[i].ID) && this.items[i].Read) {
					return true;
				}
			}
			return false;
		},

		// websocket connect
		connect: function () {
			let wsproto = location.protocol == 'https:' ? 'wss' : 'ws';
			let ws = new WebSocket(
				wsproto + "://" + document.location.host + document.location.pathname + "api/events"
			);
			let self = this;
			ws.onmessage = function (e) {
				let response = JSON.parse(e.data);
				if (!response) {
					return;
				}
				// new messages
				if (response.Type == "new" && response.Data) {
					if (!self.searching) {
						if (self.start < 1) {
							self.items.unshift(response.Data);
							if (self.items.length > self.limit) {
								self.items.pop();
							}
						} else {
							self.start++;
						}
					}
					self.total++;
					self.unread++;
					let from = response.Data.From != null ? response.Data.From.Address : '[unknown]';
					self.browserNotify("New mail from: " + from, response.Data.Subject);
				} else if (response.Type == "prune") {
					// messages have been deleted, reload messages to adjust
					self.scrollInPlace = true;
					self.loadMessages();
				}
			}

			ws.onopen = function () {
				self.isConnected = true;
				self.loadMessages();
			}

			ws.onclose = function (e) {
				self.isConnected = false;

				setTimeout(function () {
					self.connect(); // reconnect
				}, 1000);
			}

			ws.onerror = function (err) {
				ws.close();
			}
		},

		getPrimaryEmailTo: function (message) {
			for (let i in message.To) {
				return message.To[i].Address;
			}

			return '[ Undisclosed recipients ]';
		},

		getRelativeCreated: function (message) {
			let d = new Date(message.Created)
			return moment(d).fromNow().toString();
		},

		browserNotify: function (title, message) {
			if (!("Notification" in window)) {
				return;
			}

			if (Notification.permission === "granted") {
				let b = message.Subject;
				let options = {
					body: message,
					icon: 'mailpit.png'
				}
				new Notification(title, options);
			}
		},

		requestNotifications: function () {
			// check if the browser supports notifications
			if (!("Notification" in window)) {
				alert("This browser does not support desktop notification");
			}

			// we need to ask the user for permission
			else if (Notification.permission !== "denied") {
				let self = this;
				Notification.requestPermission().then(function (permission) {
					// if the user accepts, let's create a notification
					if (permission === "granted") {
						self.browserNotify("Notifications enabled", "You will receive notifications when new mails are received.");
						self.notificationsEnabled = true;
					}
				});
			}
		},

		toggleSelected: function (e, id) {
			e.preventDefault();

			if (this.isSelected(id)) {
				this.selected = this.selected.filter(function (ele) {
					return ele != id;
				});
			} else {
				this.selected.push(id);
			}
		},

		selectRange: function (e, id) {
			e.preventDefault();

			let selecting = false;
			let lastSelected = this.selected.length > 0 && this.selected[this.selected.length - 1];
			if (lastSelected == id) {
				this.selected = this.selected.filter(function (ele) {
					return ele != id;
				});
				return;
			}

			if (lastSelected === false) {
				this.selected.push(id);
				return;
			}

			for (let d of this.items) {
				if (selecting) {
					if (!this.isSelected(d.ID)) {
						this.selected.push(d.ID);
					}
					if (d.ID == lastSelected || d.ID == id) {
						// reached backwards select
						break;
					}
				} else if (d.ID == id || d.ID == lastSelected) {
					if (!this.isSelected(d.ID)) {
						this.selected.push(d.ID);
					}
					selecting = true;
				}
			}
		},

		isSelected: function (id) {
			return this.selected.indexOf(id) != -1;
		},

		loadInfo: function (e) {
			e.preventDefault();
			let self = this;
			self.get('api/v1/info', false, function (response) {
				self.appInfo = response.data;
				self.modal('AppInfoModal').show();
			});
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
			<a class="btn btn-outline-light me-4 px-3" href="#" v-on:click="message = false" title="Return to messages">
				<i class="bi bi-arrow-return-left"></i>
			</a>
			<button class="btn btn-outline-light me-2" title="Mark unread" v-on:click="markUnread">
				<i class="bi bi-eye-slash"></i> <span class="d-none d-md-inline">Mark unread</span>
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
			<a :href="'api/v1/message/' + message.ID + '/raw?dl=1'" class="btn btn-outline-light me-2 float-end"
				title="Download message">
				<i class="bi bi-file-arrow-down-fill"></i> <span class="d-none d-md-inline">Download</span>
			</a>
		</div>

		<div class="col col-md-9 col-lg-5" v-if="!message">
			<form v-on:submit="doSearch">
				<div class="input-group">
					<a class="navbar-brand d-md-none" href="#" v-on:click="reloadMessages">
						<img src="mailpit.svg" alt="Mailpit">
						<span v-if="!total" class="ms-2">Mailpit</span>
					</a>
					<div v-if="total" class="d-flex bg-white border rounded-start flex-fill position-relative">
						<input type="text" class="form-control border-0" v-model.trim="search"
							placeholder="Search mailbox">
						<span class="btn btn-link position-absolute end-0 text-muted" v-if="search"
							v-on:click="resetSearch"><i class="bi bi-x-circle"></i></span>
					</div>
					<button v-if="total" class="btn btn-outline-light" type="submit"><i
							class="bi bi-search"></i></button>
				</div>
			</form>
		</div>
		<div class="col-12 col-lg-5 text-end mt-2 mt-lg-0" v-if="!message && total">
			<button v-if="total" class="btn btn-danger float-start d-md-none me-2" data-bs-toggle="modal"
				data-bs-target="#DeleteAllModal" title="Delete all messages">
				<i class="bi bi-trash-fill"></i>
			</button>

			<button v-if="unread" class="btn btn-light float-start d-md-none" data-bs-toggle="modal"
				data-bs-target="#MarkAllReadModal" title="Mark all read">
				<i class="bi bi-check2-square"></i>
			</button>

			<select v-model="limit" v-on:change="loadMessages" class="form-select form-select-sm d-inline w-auto me-2"
				v-if="!searching">
				<option value="25">25</option>
				<option value="50">50</option>
				<option value="100">100</option>
				<option value="200">200</option>
			</select>
			<span v-if="searching">
				<b>{{ formatNumber(items.length) }} results</b>
			</span>
			<span v-else>
				<small>
					{{ formatNumber(start + 1) }}-{{ formatNumber(start + items.length) }} <small>of</small> {{
							formatNumber(total)
					}}
				</small>
				<button class="btn btn-outline-light ms-2 me-1" :disabled="!canPrev" v-on:click="viewPrev"
					v-if="!searching" :title="'View previous ' + limit + ' messages'">
					<i class="bi bi-caret-left-fill"></i>
				</button>
				<button class="btn btn-outline-light" :disabled="!canNext" v-on:click="viewNext" v-if="!searching"
					:title="'View next ' + limit + ' messages'">
					<i class="bi bi-caret-right-fill"></i>
				</button>
			</span>
		</div>
	</div>
	<div class="row flex-fill" style="min-height:0">
		<div class="d-none d-md-block col-lg-2 col-md-3 mh-100 position-relative" style="overflow-y: auto;">
			<ul class="list-unstyled mt-3 mb-5">
				<li v-if="isConnected" title="Messages will auto-load" class="mb-3 text-muted">
					<i class="bi bi-power text-success"></i>
					Connected
				</li>
				<li v-else title="You need to manually refresh your mailbox" class="mb-3">
					<i class="bi bi-power text-danger"></i>
					Disconnected
				</li>
				<li class="mb-5">
					<a class="position-relative ps-0" href="#" v-on:click="reloadMessages">
						<i class="bi bi-envelope me-1" v-if="isConnected"></i>
						<i class="bi bi-arrow-clockwise me-1" v-else></i>
						Inbox
						<span class="badge rounded-pill text-bg-primary ms-1" title="Unread messages" v-if="unread">
							{{ formatNumber(unread) }}
						</span>
					</a>
				</li>
				<li class="my-3" v-if="!message && unread && !selected.length">
					<a href="#" data-bs-toggle="modal" data-bs-target="#MarkAllReadModal">
						<i class="bi bi-eye-fill"></i>
						Mark all read
					</a>
				</li>
				<li class="my-3" v-if="!message && total && !selected.length">
					<a href="#" data-bs-toggle="modal" data-bs-target="#DeleteAllModal">
						<i class="bi bi-trash-fill me-1 text-danger"></i>
						Delete all
					</a>
				</li>

				<li class="my-3" v-if="selected.length > 0">
					<b class="me-2">Selected {{ selected.length }}</b>
					<button class="btn btn-sm text-muted" v-on:click="selected = []" title="Unselect messages"><i
							class="bi bi-x-circle"></i></button>
				</li>
				<li class="my-3 ms-2" v-if="selected.length > 0 && selectedHasUnread()">
					<a href="#" v-on:click="markSelectedRead">
						<i class="bi bi-eye-fill"></i>
						Mark read
					</a>
				</li>
				<li class="my-3 ms-2" v-if="selected.length > 0 && selectedHasRead()">
					<a href="#" v-on:click="markSelectedUnread">
						<i class="bi bi-eye-slash"></i>
						Mark unread
					</a>
				</li>
				<li class="my-3 ms-2" v-if="total && selected.length > 0">
					<a href="#" v-on:click="deleteMessages">
						<i class="bi bi-trash-fill me-1 text-danger"></i>
						Delete
					</a>
				</li>

				<li class="my-3" v-if="notificationsSupported && !notificationsEnabled">
					<a href="#" data-bs-toggle="modal" data-bs-target="#EnableNotificationsModal"
						title="Enable browser notifications">
						<i class="bi bi-bell"></i>
						Enable alerts
					</a>
				</li>
				<li class="mt-5 position-fixed bottom-0 bg-white py-2 text-muted">
					<a href="#" class="text-muted" v-on:click="loadInfo">
						<i class="bi bi-info-circle-fill"></i>
						About
					</a>
				</li>
			</ul>
		</div>

		<div class="col-lg-10 col-md-9 mh-100 pe-0">
			<div class="mh-100" style="overflow-y: auto;" :class="message ? 'd-none' : ''" id="message-page">
				<div class="list-group my-2" v-if="items.length">
					<a v-for="message in items" :href="'#' + message.ID"
						v-on:click.ctrl="toggleSelected($event, message.ID)"
						v-on:click.shift="selectRange($event, message.ID)"
						class="row message d-flex small list-group-item list-group-item-action border-start-0 border-end-0"
						:class="message.Read ? 'read' : '', isSelected(message.ID) ? 'selected' : ''">
						<div class="col-lg-3">
							<div class="d-lg-none float-end text-muted text-nowrap small">
								<i class="bi bi-paperclip h6 me-1" v-if="message.Attachments"></i>
								{{ getRelativeCreated(message) }}
							</div>
							<div class="text-truncate d-lg-none privacy">
								<span v-if="message.From" :title="message.From.Address">{{ message.From.Name ?
										message.From.Name : message.From.Address
								}}</span>
							</div>
							<div class="text-truncate d-none d-lg-block privacy">
								<b v-if="message.From" :title="message.From.Address">{{ message.From.Name ?
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
						<div class="col-lg-6 mt-2 mt-lg-0">
							<b>{{ message.Subject != "" ? message.Subject : "[ no subject ]" }}</b>
						</div>
						<div class="d-none d-lg-block col-1 small text-end text-muted">
							<i class="bi bi-paperclip float-start h6" v-if="message.Attachments"></i>
							{{ getFileSize(message.Size) }}
						</div>
						<div class="d-none d-lg-block col-2 small text-end text-muted">
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

			<Message v-if="message" :message="message"></Message>
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
	<div class="modal fade" id="MarkAllReadModal" tabindex="-1" aria-labelledby="MarkAllReadModalLabel"
		aria-hidden="true">
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
					<button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
					<button type="button" class="btn btn-primary" data-bs-dismiss="modal"
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
						<div class="col-sm-6">
							<a class="btn btn-primary w-100" href="https://github.com/axllent/mailpit" target="_blank">
								<i class="bi bi-github"></i>
								Github
								<i class="bi bi-box-arrow-up-right"></i>
							</a>
						</div>
						<div class="col-sm-6">
							<a class="btn btn-primary w-100" href="https://github.com/axllent/mailpit/wiki"
								target="_blank">
								Documentation
								<i class="bi bi-box-arrow-up-right"></i>
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
</template>
