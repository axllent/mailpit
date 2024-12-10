<script>
import AboutMailpit from '../components/AboutMailpit.vue'
import AjaxLoader from '../components/AjaxLoader.vue'
import CommonMixins from '../mixins/CommonMixins'
import Message from '../components/message/Message.vue'
import Release from '../components/message/Release.vue'
import Screenshot from '../components/message/Screenshot.vue'
import { mailbox } from '../stores/mailbox'
import { pagination } from '../stores/pagination'
import dayjs from 'dayjs'

export default {
	mixins: [CommonMixins],

	// global event bus to handle message status changes
	inject: ["eventBus"],

	components: {
		AboutMailpit,
		AjaxLoader,
		Message,
		Screenshot,
		Release,
	},

	data() {
		return {
			mailbox,
			pagination,
			message: false,
			errorMessage: false,
			apiSideNavURI: false,
			apiSideNavParams: URLSearchParams,
			apiIsMore: true,
			messagesList: [],
			liveLoaded: 0, // the number new messages prepended tp messageList
			scrollLoading: false,
			canLoadMore: true,
		}
	},

	watch: {
		$route(to, from) {
			this.loadMessage()
		},
	},

	created() {
		const relativeTime = require('dayjs/plugin/relativeTime')
		dayjs.extend(relativeTime)

		this.initLoadMoreAPIParams()
	},

	mounted() {
		this.loadMessage()

		this.messagesList = JSON.parse(JSON.stringify(this.mailbox.messages))
		if (!this.messagesList.length) {
			this.loadMore()
		}

		this.refreshUI()

		// subscribe to events
		this.eventBus.on("new", this.handleWSNew)
		this.eventBus.on("update", this.handleWSUpdate)
		this.eventBus.on("delete", this.handleWSDelete)
		this.eventBus.on("truncate", this.handleWSTruncate)
	},

	unmounted() {
		// unsubscribe from events
		this.eventBus.off("new", this.handleWSNew)
		this.eventBus.off("update", this.handleWSUpdate)
		this.eventBus.off("delete", this.handleWSDelete)
		this.eventBus.off("truncate", this.handleWSTruncate)
	},

	computed: {
		// get current message read status
		isRead() {
			const l = this.messagesList.length
			if (!this.message || !l) {
				return true
			}

			let id = false
			for (x = 0; x < l; x++) {
				if (this.messagesList[x].ID == this.message.ID) {
					return this.messagesList[x].Read
				}
			}

			return true
		},

		// get the previous message ID
		previousID() {
			const l = this.messagesList.length
			if (!this.message || !l) {
				return false
			}

			let id = false
			for (x = 0; x < l; x++) {
				if (this.messagesList[x].ID == this.message.ID) {
					return id
				}
				id = this.messagesList[x].ID
			}

			return false
		},

		// get the next message ID
		nextID() {
			const l = this.messagesList.length
			if (!this.message || !l) {
				return false
			}

			let id = false
			for (x = l - 1; x > 0; x--) {
				if (this.messagesList[x].ID == this.message.ID) {
					return id
				}
				id = this.messagesList[x].ID
			}

			return id
		}
	},

	methods: {
		loadMessage() {
			this.message = false
			const uri = this.resolve('/api/v1/message/' + this.$route.params.id)
			this.get(uri, false, (response) => {
				this.errorMessage = false
				const d = response.data

				// update read status in case websockets is not working
				this.handleWSUpdate({ 'ID': d.ID, Read: true })

				// replace inline images embedded as inline attachments
				if (d.HTML && d.Inline) {
					for (let i in d.Inline) {
						let a = d.Inline[i]
						if (a.ContentID != '') {
							d.HTML = d.HTML.replace(
								new RegExp('(=["\']?)(cid:' + a.ContentID + ')(["|\'|\\s|\\/|>|;])', 'g'),
								'$1' + this.resolve('/api/v1/message/' + d.ID + '/part/' + a.PartID) + '$3'
							)
						}
						if (a.FileName.match(/^[a-zA-Z0-9\_\-\.]+$/)) {
							// some old email clients use the filename
							d.HTML = d.HTML.replace(
								new RegExp('(=["\']?)(' + a.FileName + ')(["|\'|\\s|\\/|>|;])', 'g'),
								'$1' + this.resolve('/api/v1/message/' + d.ID + '/part/' + a.PartID) + '$3'
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
								'$1' + this.resolve('/api/v1/message/' + d.ID + '/part/' + a.PartID) + '$3'
							)
						}
						if (a.FileName.match(/^[a-zA-Z0-9\_\-\.]+$/)) {
							// some old email clients use the filename
							d.HTML = d.HTML.replace(
								new RegExp('(=["\']?)(' + a.FileName + ')(["|\'|\\s|\\/|>|;])', 'g'),
								'$1' + this.resolve('/api/v1/message/' + d.ID + '/part/' + a.PartID) + '$3'
							)
						}
					}
				}

				this.message = d

				this.$nextTick(() => {
					this.scrollSidebarToCurrent()
				})
			},
				(error) => {
					this.errorMessage = true
					if (error.response && error.response.data) {
						if (error.response.data.Error) {
							this.errorMessage = error.response.data.Error
						} else {
							this.errorMessage = error.response.data
						}
					} else if (error.request) {
						// The request was made but no response was received
						this.errorMessage = 'Error sending data to the server. Please refresh the page.'
					} else {
						// Something happened in setting up the request that triggered an Error
						this.errorMessage = error.message
					}
				})
		},

		// UI refresh ticker to adjust relative times
		refreshUI() {
			window.setTimeout(() => {
				this.$forceUpdate()
				this.refreshUI()
			}, 30000)
		},

		// handler for websocket new messages
		handleWSNew(data) {
			// do not add when searching or >= 100 new messages have been received
			if (this.mailbox.searching || this.liveLoaded >= 100) {
				return
			}

			this.liveLoaded++
			this.messagesList.unshift(data)
		},

		// handler for websocket message updates
		handleWSUpdate(data) {
			for (let x = 0; x < this.messagesList.length; x++) {
				if (this.messagesList[x].ID == data.ID) {
					// update message
					this.messagesList[x] = { ...this.messagesList[x], ...data }
					return
				}
			}
		},

		// handler for websocket message deletion
		handleWSDelete(data) {
			for (let x = 0; x < this.messagesList.length; x++) {
				if (this.messagesList[x].ID == data.ID) {
					// remove message from the list
					this.messagesList.splice(x, 1)
					return
				}
			}
		},

		// handler for websocket message truncation
		handleWSTruncate() {
			// all messages gone, go to inbox
			this.$router.push('/')
		},

		// return whether the sidebar is visible
		sidebarVisible() {
			return this.$refs.MessageList.offsetParent != null
		},

		// scroll sidenav to current message if found
		scrollSidebarToCurrent() {
			const cont = document.getElementById('MessageList')
			if (!cont) {
				return
			}
			const c = cont.querySelector('.router-link-active')
			if (c) {
				const outer = cont.getBoundingClientRect()
				const li = c.getBoundingClientRect()
				if (outer.top > li.top || outer.bottom < li.bottom) {
					c.scrollIntoView({ behavior: "smooth", block: "center", inline: "nearest" })
				}
			}
		},

		scrollHandler(e) {
			if (!this.canLoadMore || this.scrollLoading) {
				return
			}

			const { scrollTop, offsetHeight, scrollHeight } = e.target
			if ((scrollTop + offsetHeight + 150) >= scrollHeight) {
				this.loadMore()
			}
		},

		loadMore() {
			if (this.messagesList.length) {
				// get last created timestamp
				const oldest = this.messagesList[this.messagesList.length - 1].Created
				// if set append `before=<ts>` 
				this.apiSideNavParams.set('before', oldest)
			}

			this.scrollLoading = true

			this.get(this.apiSideNavURI, this.apiSideNavParams, (response) => {
				if (response.data.messages.length) {
					this.messagesList.push(...response.data.messages)
				} else {
					this.canLoadMore = false
				}
				this.$nextTick(() => {
					this.scrollLoading = false
				})
			}, null, true)
		},

		initLoadMoreAPIParams() {
			let apiURI = this.resolve(`/api/v1/messages`)
			let p = {}

			if (mailbox.searching) {
				apiURI = this.resolve(`/api/v1/search`)
				p.query = mailbox.searching
			}

			if (pagination.limit != pagination.defaultLimit) {
				p.limit = pagination.limit.toString()
			}

			this.apiSideNavURI = apiURI

			this.apiSideNavParams = new URLSearchParams(p)
		},

		getRelativeCreated(message) {
			const d = new Date(message.Created)
			return dayjs(d).fromNow()
		},

		getPrimaryEmailTo(message) {
			for (let i in message.To) {
				return message.To[i].Address
			}

			return '[ Undisclosed recipients ]'
		},

		isActive(id) {
			return this.message.ID == id
		},

		toTagUrl(t) {
			if (t.match(/ /)) {
				t = `"${t}"`
			}
			const p = {
				q: 'tag:' + t
			}
			if (pagination.limit != pagination.defaultLimit) {
				p.limit = pagination.limit.toString()
			}
			const params = new URLSearchParams(p)
			return '/search?' + params.toString()
		},

		downloadMessageBody(str, ext) {
			const dl = document.createElement('a')
			dl.href = "data:text/plain," + encodeURIComponent(str)
			dl.target = '_blank'
			dl.download = this.message.ID + '.' + ext
			dl.click()
		},

		screenshotMessageHTML() {
			this.$refs.ScreenshotRef.initScreenshot()
		},

		// toggle current message read status
		toggleRead() {
			if (!this.message) {
				return false
			}
			const read = !this.isRead

			const ids = [this.message.ID]
			const uri = this.resolve('/api/v1/messages')
			this.put(uri, { 'Read': read, 'IDs': ids }, () => {
				if (!this.sidebarVisible()) {
					return this.goBack()
				}

				// manually update read status in case websockets is not working
				this.handleWSUpdate({ 'ID': this.message.ID, Read: read })
			})
		},

		deleteMessage() {
			const ids = [this.message.ID]
			const uri = this.resolve('/api/v1/messages')
			// calculate next ID before deletion to prevent WS race
			const goToID = this.nextID ? this.nextID : this.previousID

			this.delete(uri, { 'IDs': ids }, () => {
				if (!this.sidebarVisible()) {
					return this.goBack()
				}
				if (goToID) {
					return this.$router.push('/view/' + goToID)
				}

				return this.goBack()
			})
		},

		// return to mailbox or search based on origin
		goBack() {
			mailbox.lastMessage = this.$route.params.id

			if (mailbox.searching) {
				const p = {
					q: mailbox.searching
				}
				if (pagination.start > 0) {
					p.start = pagination.start.toString()
				}
				if (pagination.limit != pagination.defaultLimit) {
					p.limit = pagination.limit.toString()
				}
				this.$router.push('/search?' + new URLSearchParams(p).toString())
			} else {
				const p = {}
				if (pagination.start > 0) {
					p.start = pagination.start.toString()
				}
				if (pagination.limit != pagination.defaultLimit) {
					p.limit = pagination.limit.toString()
				}
				this.$router.push('/?' + new URLSearchParams(p).toString())
			}
		},

		reloadWindow() {
			location.reload()
		},

		initReleaseModal() {
			this.modal('ReleaseModal').show()
			window.setTimeout(() => {
				// delay to allow elements to load / focus
				this.$refs.ReleaseRef.initTags()
				document.querySelector('#ReleaseModal input[role="combobox"]').focus()
			}, 500)
		},
	}
}
</script>

<template>
	<div class="navbar navbar-expand-lg navbar-dark row flex-shrink-0 bg-primary text-white d-print-none">
		<div class="d-none d-xl-block col-xl-3 col-auto pe-0">
			<RouterLink to="/" class="navbar-brand text-white me-0" @click="pagination.start = 0">
				<img :src="resolve('/mailpit.svg')" alt="Mailpit">
				<span class="ms-2 d-none d-sm-inline">Mailpit</span>
			</RouterLink>
		</div>
		<div class="col col-xl-5" v-if="!errorMessage">
			<button @click="goBack()" class="btn btn-outline-light me-3 d-xl-none" title="Return to messages">
				<i class="bi bi-arrow-return-left"></i>
				<span class="ms-2 d-none d-lg-inline">Back</span>
			</button>
			<button class="btn btn-outline-light me-1 me-sm-2" title="Mark unread" v-on:click="toggleRead()">
				<i class="bi bi-eye-slash me-md-2" :class="isRead ? 'bi-eye-slash' : 'bi-eye'"></i>
				<span class="d-none d-md-inline">Mark <template v-if="isRead">un</template>read</span>
			</button>
			<button class="btn btn-outline-light me-1 me-sm-2" title="Release message"
				v-if="mailbox.uiConfig.MessageRelay && mailbox.uiConfig.MessageRelay.Enabled"
				v-on:click="initReleaseModal()">
				<i class="bi bi-send"></i> <span class="d-none d-md-inline">Release</span>
			</button>
			<button class="btn btn-outline-light me-1 me-sm-2" title="Delete message" v-on:click="deleteMessage()">
				<i class="bi bi-trash-fill"></i> <span class="d-none d-md-inline">Delete</span>
			</button>
		</div>
		<div class="col-auto col-lg-4 col-xl-4 text-end" v-if="!errorMessage">
			<div class="dropdown d-inline-block" id="DownloadBtn">
				<button type="button" class="btn btn-outline-light dropdown-toggle" data-bs-toggle="dropdown"
					aria-expanded="false">
					<i class="bi bi-file-arrow-down-fill"></i>
					<span class="d-none d-md-inline ms-1">Download</span>
				</button>
				<ul class="dropdown-menu dropdown-menu-end">
					<li>
						<a :href="resolve('/api/v1/message/' + message.ID + '/raw?dl=1')" class="dropdown-item"
							title="Message source including headers, body and attachments">
							Raw message
						</a>
					</li>
					<li v-if="message.HTML">
						<button v-on:click="downloadMessageBody(message.HTML, 'html')" class="dropdown-item">
							HTML body
						</button>
					</li>
					<li v-if="message.HTML">
						<button class="dropdown-item" @click="screenshotMessageHTML()">
							HTML screenshot
						</button>
					</li>
					<li v-if="message.Text">
						<button v-on:click="downloadMessageBody(message.Text, 'txt')" class="dropdown-item">
							Text body
						</button>
					</li>
					<template v-if="message.Attachments && message.Attachments.length">
						<li>
							<hr class="dropdown-divider">
						</li>
						<li>
							<h6 class="dropdown-header">
								Attachments
							</h6>
						</li>
						<li v-for="part in message.Attachments">
							<RouterLink :to="'/api/v1/message/' + message.ID + '/part/' + part.PartID"
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
							</RouterLink>
						</li>
					</template>
					<template v-if="message.Inline && message.Inline.length">
						<li>
							<hr class="dropdown-divider">
						</li>
						<li>
							<h6 class="dropdown-header">
								Inline image<span v-if="message.Inline.length > 1">s</span>
							</h6>
						</li>
						<li v-for="part in message.Inline">
							<RouterLink :to="'/api/v1/message/' + message.ID + '/part/' + part.PartID"
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
							</RouterLink>
						</li>
					</template>
				</ul>
			</div>

			<RouterLink :to="'/view/' + previousID" class="btn btn-outline-light ms-1 ms-sm-2 me-1"
				:class="previousID ? '' : 'disabled'" title="View previous message">
				<i class="bi bi-caret-left-fill"></i>
			</RouterLink>
			<RouterLink :to="'/view/' + nextID" class="btn btn-outline-light" :class="nextID ? '' : 'disabled'">
				<i class="bi bi-caret-right-fill" title="View next message"></i>
			</RouterLink>
		</div>
	</div>

	<div class="row flex-fill" style="min-height:0">
		<div class="d-none d-xl-flex col-xl-3 h-100 flex-column">
			<div class="text-center badge text-bg-primary py-2 my-2 w-100" v-if="mailbox.uiConfig.Label">
				<div class="text-truncate fw-normal">
					{{ mailbox.uiConfig.Label }}
				</div>
			</div>

			<div class="list-group my-2" :class="mailbox.uiConfig.Label ? 'mt-0' : ''">
				<button @click="goBack()" class="list-group-item list-group-item-action">
					<i class="bi bi-arrow-return-left me-1"></i>
					<span class="ms-1">
						Return to
						<template v-if="mailbox.searching">search</template>
						<template v-else>inbox</template>
					</span>
					<span class="badge rounded-pill ms-1 float-end text-bg-secondary" title="Unread messages"
						v-if="mailbox.unread && !errorMessage">
						{{ formatNumber(mailbox.unread) }}
					</span>
				</button>
			</div>

			<div class="flex-grow-1 overflow-y-auto px-1 me-n1" id="MessageList" ref="MessageList"
				@scroll="scrollHandler">
				<button v-if="liveLoaded >= 100" class="w-100 alert alert-warning small" @click="reloadWindow()">
					Reload to see newer messages
				</button>
				<template v-if="messagesList && messagesList.length">
					<div class="list-group">
						<RouterLink v-for="message in messagesList" :to="'/view/' + message.ID" :key="message.ID"
							:id="message.ID"
							class="row gx-1 message d-flex small list-group-item list-group-item-action"
							:class="message.Read ? 'read' : '', isActive(message.ID) ? 'active' : ''">
							<div class="col overflow-x-hidden">
								<div class="text-truncate privacy small">
									<strong v-if="message.From" :title="'From: ' + message.From.Address">
										{{ message.From.Name ? message.From.Name : message.From.Address }}
									</strong>
								</div>
							</div>
							<div class="col-auto small">
								<i class="bi bi-paperclip h6" v-if="message.Attachments"></i>
								{{ getRelativeCreated(message) }}
							</div>
							<div class="col-12 overflow-x-hidden">
								<div class="text-truncate privacy small">
									To: {{ getPrimaryEmailTo(message) }}
									<span v-if="message.To && message.To.length > 1">
										[+{{ message.To.length - 1 }}]
									</span>
								</div>
							</div>
							<div class="col-12 overflow-x-hidden mt-1">
								<div class="text-truncates small">
									<b>{{ message.Subject != "" ? message.Subject : "[ no subject ]" }}</b>
								</div>
							</div>
							<div v-if="message.Tags.length" class="col-12">
								<RouterLink class="badge me-1" v-for="t in message.Tags" :to="toTagUrl(t)"
									v-on:click="pagination.start = 0"
									:style="mailbox.showTagColors ? { backgroundColor: colorHash(t) } : { backgroundColor: '#6c757d' }"
									:title="'Filter messages tagged with ' + t">
									{{ t }}
								</RouterLink>
							</div>
						</RouterLink>
					</div>
				</template>
			</div>

			<AboutMailpit />
		</div>

		<div class="col-xl-9 mh-100 ps-0 ps-md-2 pe-0">
			<div class="mh-100" style="overflow-y: auto;" id="message-page">
				<template v-if="errorMessage">
					<h3 class="text-center my-3">
						{{ errorMessage }}
					</h3>
				</template>
				<Message v-else-if="message" :key="message.ID" :message="message" />
			</div>
		</div>
	</div>

	<AboutMailpit modals />
	<AjaxLoader :loading="loading" />
	<Release v-if="mailbox.uiConfig.MessageRelay && message" ref="ReleaseRef" :message="message"
		@delete="deleteMessage" />
	<Screenshot v-if="message" ref="ScreenshotRef" :message="message" />
</template>
