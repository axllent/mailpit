<script>
import commonMixins from './mixins.js'
import Message from './templates/Message.vue';
import moment from 'moment'

export default {
	mixins: [commonMixins],
	components: {
		Message
	},
	data() {
		return {
			currentPath: window.location.hash,
			mailbox: "catchall",
			items: [],
			limit: 50,
			total: 0,
			unread: 0,
			start: 0,
			search: "",
			searching: false,
			isConnected: false,
			scrollInPlace: false,
			message: false,
			notificationsSupported: false,
			notificationsEnabled: false
		}
	},
	watch: {
		currentPath(v, old) {
			if (v && v.match(/^[a-z0-9]+-[a-z0-9]+-[a-z0-9]+-[a-z0-9]+-[a-z0-9]+$/)) {
				this.openMessage();
			} else {
				this.message = false;
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

		this.connect();
		this.loadMessages();
	},
	methods: {
		loadMessages: function () {
            let self = this;
            let params = {};

            let uri = 'api/'+self.mailbox+'/messages';
            if (self.search) {
                self.searching = true;
				self.items = [];
                uri = 'api/'+self.mailbox+'/search'
                self.start = 0; // search is displayed on one page
                params['query'] = self.search;
            } else {
				self.searching = false;
                params['limit'] = self.limit;
                if (self.start > 0) {
                    params['start'] = self.start;
                }
            }

			self.get(uri, params, function(response){
				self.total = response.data.total;
				self.unread = response.data.unread;
				self.count = response.data.count;
				self.start = response.data.start;
				self.items = response.data.items;

				if (!self.scrollInPlace) {
					let mp = document.getElementById('message-page');
					if (mp) {
						mp.scrollTop = 0;
					}
				}

				self.scrollInPlace = false
			});
        },

		doSearch: function(e) {
			e.preventDefault();
			this.loadMessages();
		},

		reloadMessages: function() {
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

		openMessage: function(id) {
			let self = this;
            let params = {};

            let uri = 'api/' +  self.mailbox + '/' + self.currentPath
			self.get(uri, params, function(response) {
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
								new RegExp('cid:'+a.ContentID, 'g'), 
								window.location.origin+'/api/'+self.mailbox+'/'+d.ID+'/part/'+a.PartID
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
								new RegExp('cid:'+a.ContentID, 'g'), 
								window.location.origin+'/api/'+self.mailbox+'/'+d.ID+'/part/'+a.PartID
							);
						}
					}
				}

				self.message = d;
			});
		},

		deleteAll: function() {
			let self = this;
			let uri = 'api/' +  self.mailbox + '/delete'
			self.get(uri, false, function(response) {
				self.reloadMessages();
			});
		},

		deleteOne: function() {
			let self = this;
			if (!self.message) {
				return false;
			}
			let uri = 'api/' +  self.mailbox + '/' + self.message.ID + '/delete'
			self.get(uri, false, function(response) {
				window.location.hash = "";
				self.scrollInPlace = true;
				self.loadMessages();

			});
		},

		markUnread: function() {
			let self = this;
			if (!self.message) {
				return false;
			}
			let uri = 'api/' +  self.mailbox + '/' + self.message.ID + '/unread'
			self.get(uri, false, function(response) {
				window.location.hash = "";
				self.scrollInPlace = true;
				self.loadMessages();
			});
		},

		markAllRead: function() {
			let self = this;
			let uri = 'api/' +  self.mailbox + '/read'
			self.get(uri, false, function(response) {
				window.location.hash = "";
				self.scrollInPlace = true;
				self.loadMessages();
			});
		},

		// websocket connect
        connect: function () {
            let wsproto = location.protocol == 'https:' ? 'wss' : 'ws';
            let ws = new WebSocket(wsproto + "://" + document.location.host + "/api/"+this.mailbox+"/events");
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
					self.browserNotify("New mail from: " + response.Data.From.Address, response.Data.Subject);
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

		getPrimaryEmailTo: function(message) {
			for (let i in message.To) {
				return message.To[i].Address;
			}

			return '[ Unknown ]';
		},

		getRelativeCreated: function(message) {
            let d = new Date(message.Created)
            return moment(d).fromNow().toString();
        },

		browserNotify: function(title, message) {
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

		requestNotifications: function() {
			// check if the browser supports notifications
			if (!("Notification" in window)) {
				alert("This browser does not support desktop notification");
			}

			// we need to ask the user for permission
			else if (Notification.permission !== "denied") {
				let self = this;
				Notification.requestPermission().then(function (permission) {
				// If the user accepts, let's create a notification
					if (permission === "granted") {
						self.browserNotify("Notifications enabled", "You will receive notifications when new mails are received.");
						self.notificationsEnabled = true;
					}
				});
			}
		}

	}
}
</script>

<template>
	<div class="navbar navbar-expand-lg navbar-light row flex-shrink-0 bg-light">
		<div class="col-lg-2 col-md-3 col-auto">
			<a class="navbar-brand" href="#" v-on:click="reloadMessages">
				<img src="mailpit.svg" alt="Mailpit">
				<span class="d-none d-md-inline-block ms-2">Mailpit</span>
			</a>
		</div>
		
		<div class="col col-md-9 col-lg-8" v-if="message">
			<a class="btn btn-outline-secondary me-4 px-3" href="#" v-on:click="message=false" title="Return to messages">
				<i class="bi bi-arrow-return-left"></i>
			</a>
			<button class="btn btn-outline-secondary me-2" title="Delete message" v-on:click="deleteOne">
				<i class="bi bi-trash-fill"></i>
			</button>
			<button class="btn btn-outline-secondary me-2" title="Mark unread" v-on:click="markUnread">
				<i class="bi bi-envelope"></i>
			</button>
			<a :href="'api/' + mailbox + '/' + message.ID + '/source?dl=1'" class="btn btn-outline-secondary me-2" title="Download message">
				<i class="bi bi-file-arrow-down-fill"></i>
			</a>
		</div>

		<div class="col col-md-9 col-lg-5" v-if="!message && total">
			<form v-on:submit="doSearch">
				<div class="input-group">
					<input type="text" class="form-control" v-model.trim="search" placeholder="Search mailbox">
					<button class="btn btn-outline-secondary" type="submit"><i class="bi bi-search"></i></button>
				</div>
			</form>
		</div>
		<div class="col-12 col-lg-5 text-end" v-if="!message && total">
			<select v-model="limit" v-on:change="loadMessages"
				class="form-select form-select-sm d-inline w-auto me-1" v-if="!searching">
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
					<b>{{ formatNumber(start + 1) }}-{{ formatNumber(start + items.length) }}</b> of <b>{{ formatNumber(total) }}</b>
				</small>
				<button class="btn btn-outline-secondary ms-3 me-1" :disabled="!canPrev" v-on:click="viewPrev"
					v-if="!searching">
					<i class="bi bi-caret-left-fill"></i>
				</button>
				<button class="btn btn-outline-secondary" :disabled="!canNext" v-on:click="viewNext" v-if="!searching">
					<i class="bi bi-caret-right-fill"></i>
				</button>
			</span>
		</div>
	</div>
	<div class="row flex-fill" style="min-height:0">
		<div class="d-none d-md-block col-lg-2 col-md-3 mh-100 position-relative" style="overflow-y: auto;">
			<ul class="list-unstyled mt-3 mb-5">
				<li v-if="isConnected" title="Messages will auto-load" class="mb-2">
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
						<span class="position-absolute mt-2 ms-4 start-100 translate-middle badge rounded-pill text-bg-secondary" title="Unread messages" v-if="unread">
							{{ formatNumber(unread) }}
						</span>
					</a>
				</li>
				<li class="my-3" v-if="unread">
					<a href="#" data-bs-toggle="modal" data-bs-target="#MarkAllReadModal">
						<i class="bi bi-check2-square"></i>
						Mark all read
					</a>
				</li>
				<li class="my-3" v-if="total">
					<a href="#" data-bs-toggle="modal" data-bs-target="#DeleteAllModal">
						<i class="bi bi-trash-fill me-1 text-danger"></i>
						Delete all
					</a>
				</li>
				<li class="my-3" v-if="notificationsSupported && !notificationsEnabled">
					<a href="#" data-bs-toggle="modal" data-bs-target="#EnableNotificationsModal" title="Enable browser notifications">
						<i class="bi bi-bell"></i>
						Enable alerts
					</a>
				</li>
				<li class="mt-5 position-fixed bottom-0">
					<a href="https://github.com/axllent/mailpit" target="_blank" class="text-muted w-100 d-block bg-white my-3">
						<i class="bi bi-github"></i>
						GitHub
					</a>
				</li>
			</ul>
		</div>

		<div class="col-lg-10 col-md-9 mh-100 pe-0">
			<div class="mh-100" style="overflow-y: auto;" :class="message ? 'd-none':''" id="message-page">
				<div class="list-group" v-if="items.length">
					<a v-for="message in items" :href="'#'+message.ID" class="row message d-flex small list-group-item list-group-item-action"
						:class="message.Read ? 'read':''" XXXv-on:click="openMessage(message)">
						<div class="col-md-3">
							<div class="d-md-none float-end text-muted text-nowrap small">
								<i class="bi bi-paperclip h6 me-1" v-if="message.Attachments"></i>
								{{ getRelativeCreated(message) }}
							</div>

							<div class="text-truncate d-md-none">
								<span v-if="message.From" :title="message.From.Address">{{ message.From.Name ? message.From.Name : message.From.Address }}</span>
							</div> 
							<div class="text-truncate d-none d-md-block">
								<b v-if="message.From" :title="message.From.Address">{{ message.From.Name ? message.From.Name : message.From.Address }}</b>
							</div>
							<div class="d-none d-md-block text-truncate text-muted small">
								{{ getPrimaryEmailTo(message) }}
								<span v-if="message.To && message.To.length > 1">
									[+{{message.To.length - 1}}]
								</span>
							</div>
						</div>
						<div class="col-md-6 mt-2 mt-md-0">
							<b>{{ message.Subject != "" ? message.Subject : "[ no subject ]" }}</b>
						</div>
						<div class="d-none d-md-block col-1 small text-end text-muted">
							<i class="bi bi-paperclip float-start h6" v-if="message.Attachments"></i>
							{{ getFileSize(message.Size) }}
						</div>
						<div class="d-none d-md-block col-2 small text-end text-muted">
							{{ getRelativeCreated(message) }}
						</div>
					</a>
				</div>
				<div v-else class="text-muted my-3">No messages</div>
			</div>

			<Message v-if="message" :message="message" :mailbox="mailbox"></Message>
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
				<button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
				<button type="button" class="btn btn-danger" data-bs-dismiss="modal" v-on:click="deleteAll">Delete</button>
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
				<button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
				<button type="button" class="btn btn-primary" data-bs-dismiss="modal" v-on:click="markAllRead">Confirm</button>
			</div>
			</div>
		</div>
	</div>

	<!-- Modal -->
	<div class="modal fade" id="EnableNotificationsModal" tabindex="-1" aria-labelledby="EnableNotificationsModalLabel" aria-hidden="true">
		<div class="modal-dialog modal-lg">
			<div class="modal-content">
			<div class="modal-header">
				<h5 class="modal-title" id="EnableNotificationsModalLabel">Enable browser notifications?</h5>
				<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
			</div>
			<div class="modal-body">
				<p class="h4">Get browser notifications when Mailpit receives a new mail?</p>
				<p>
					Note that your browser will ask you for confirmation when you click <code>enable notifications</code>,
					and that you must have Mailpit open in a browser tab to be able to receive the notifications.
				</p>
			</div>
			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
				<button type="button" class="btn btn-primary" data-bs-dismiss="modal" v-on:click="requestNotifications">Enable notifications</button>
			</div>
			</div>
		</div>
	</div>

</template>
