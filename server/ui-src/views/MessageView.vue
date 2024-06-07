<script>
import AboutMailpit from '../components/AboutMailpit.vue'
import AjaxLoader from '../components/AjaxLoader.vue'
import CommonMixins from '../mixins/CommonMixins'
import Message from '../components/message/Message.vue'
import Release from '../components/message/Release.vue'
import Screenshot from '../components/message/Screenshot.vue'
import { mailbox } from '../stores/mailbox'
import { pagination } from '../stores/pagination'

export default {
	mixins: [CommonMixins],

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
			prevLink: false,
			nextLink: false,
			errorMessage: false,
		}
	},

	watch: {
		$route(to, from) {
			this.loadMessage()
		}
	},

	mounted() {
		this.loadMessage()
	},

	methods: {
		loadMessage: function () {
			let self = this
			this.message = false
			let uri = self.resolve('/api/v1/message/' + this.$route.params.id)
			self.get(uri, false, function (response) {
				self.errorMessage = false

				let d = response.data

				if (self.wasUnread(d.ID)) {
					mailbox.unread--
				}

				// replace inline images embedded as inline attachments
				if (d.HTML && d.Inline) {
					for (let i in d.Inline) {
						let a = d.Inline[i]
						if (a.ContentID != '') {
							d.HTML = d.HTML.replace(
								new RegExp('(=["\']?)(cid:' + a.ContentID + ')(["|\'|\\s|\\/|>|;])', 'g'),
								'$1' + self.resolve('/api/v1/message/' + d.ID + '/part/' + a.PartID) + '$3'
							)
						}
						if (a.FileName.match(/^[a-zA-Z0-9\_\-\.]+$/)) {
							// some old email clients use the filename
							d.HTML = d.HTML.replace(
								new RegExp('(=["\']?)(' + a.FileName + ')(["|\'|\\s|\\/|>|;])', 'g'),
								'$1' + self.resolve('/api/v1/message/' + d.ID + '/part/' + a.PartID) + '$3'
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
								'$1' + self.resolve('/api/v1/message/' + d.ID + '/part/' + a.PartID) + '$3'
							)
						}
						if (a.FileName.match(/^[a-zA-Z0-9\_\-\.]+$/)) {
							// some old email clients use the filename
							d.HTML = d.HTML.replace(
								new RegExp('(=["\']?)(' + a.FileName + ')(["|\'|\\s|\\/|>|;])', 'g'),
								'$1' + self.resolve('/api/v1/message/' + d.ID + '/part/' + a.PartID) + '$3'
							)
						}
					}
				}

				self.message = d

				self.detectPrevNext()
			},
				function (error) {
					self.errorMessage = true
					if (error.response && error.response.data) {
						if (error.response.data.Error) {
							self.errorMessage = error.response.data.Error
						} else {
							self.errorMessage = error.response.data
						}
					} else if (error.request) {
						// The request was made but no response was received
						self.errorMessage = 'Error sending data to the server. Please refresh the page.'
					} else {
						// Something happened in setting up the request that triggered an Error
						self.errorMessage = error.message
					}
				})
		},

		// try detect whether this message was unread based on messages listing
		wasUnread: function (id) {
			for (let m in mailbox.messages) {
				if (mailbox.messages[m].ID == id) {
					if (!mailbox.messages[m].Read) {
						mailbox.messages[m].Read = true
						return true
					}
					return false
				}
			}
		},

		detectPrevNext: function () {
			// generate the prev/next links based on current message list
			this.prevLink = false
			this.nextLink = false
			let found = false

			for (let m in mailbox.messages) {
				if (mailbox.messages[m].ID == this.message.ID) {
					found = true
				} else if (found && !this.nextLink) {
					this.nextLink = mailbox.messages[m].ID
					break
				} else {
					this.prevLink = mailbox.messages[m].ID
				}
			}
		},

		downloadMessageBody: function (str, ext) {
			let dl = document.createElement('a')
			dl.href = "data:text/plain," + encodeURIComponent(str)
			dl.target = '_blank'
			dl.download = this.message.ID + '.' + ext
			dl.click()
		},

		screenshotMessageHTML: function () {
			this.$refs.ScreenshotRef.initScreenshot()
		},

		// mark current message as read
		markUnread: function () {
			let self = this
			if (!self.message) {
				return false
			}
			let uri = self.resolve('/api/v1/messages')
			self.put(uri, { 'read': false, 'ids': [self.message.ID] }, function (response) {
				self.goBack()
			})
		},

		deleteMessage: function () {
			let self = this
			let ids = [self.message.ID]
			let uri = self.resolve('/api/v1/messages')
			self.delete(uri, { 'ids': ids }, function () {
				self.goBack()
			})
		},

		goBack: function () {
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
				const params = new URLSearchParams(p)
				this.$router.push('/search?' + params.toString())
			} else {
				const p = {}
				if (pagination.start > 0) {
					p.start = pagination.start.toString()
				}
				if (pagination.limit != pagination.defaultLimit) {
					p.limit = pagination.limit.toString()
				}
				const params = new URLSearchParams(p)
				this.$router.push('/?' + params.toString())
			}
		},

		initReleaseModal: function () {
			let self = this
			self.modal('ReleaseModal').show()
			window.setTimeout(function () {
				window.setTimeout(function () {
					// delay to allow elements to load / focus
					self.$refs.ReleaseRef.initTags()
					document.querySelector('#ReleaseModal input[role="combobox"]').focus()
				}, 500)
			}, 300)
		},
	}
}
</script>

<template>
	<div class="navbar navbar-expand-lg navbar-dark row flex-shrink-0 bg-primary text-white">
		<div class="d-none d-md-block col-xl-2 col-md-3 col-auto pe-0">
			<RouterLink to="/" class="navbar-brand text-white me-0" @click="pagination.start = 0">
				<img :src="resolve('/mailpit.svg')" alt="Mailpit">
				<span class="ms-2 d-none d-sm-inline">Mailpit</span>
			</RouterLink>
		</div>
		<div class="col col-md-4k col-lg-5 col-xl-6" v-if="!errorMessage">
			<button @click="goBack()" class="btn btn-outline-light me-3 me-sm-4 d-md-none" title="Return to messages">
				<i class="bi bi-arrow-return-left"></i>
			</button>
			<button class="btn btn-outline-light me-1 me-sm-2" title="Mark unread" v-on:click="markUnread">
				<i class="bi bi-eye-slash"></i> <span class="d-none d-md-inline">Mark unread</span>
			</button>
			<button class="btn btn-outline-light me-1 me-sm-2" title="Release message"
				v-if="mailbox.uiConfig.MessageRelay && mailbox.uiConfig.MessageRelay.Enabled"
				v-on:click="initReleaseModal">
				<i class="bi bi-send"></i> <span class="d-none d-md-inline">Release</span>
			</button>
			<button class="btn btn-outline-light me-1 me-sm-2" title="Delete message" v-on:click="deleteMessage">
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
					<template v-if="allAttachments(message).length">
						<li>
							<hr class="dropdown-divider">
						</li>
						<li>
							<h6 class="dropdown-header">
								Attachment<template v-if="allAttachments(message).length > 1">s</template>
							</h6>
						</li>
						<li v-for="part in allAttachments(message)">
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

			<RouterLink :to="'/view/' + prevLink" class="btn btn-outline-light ms-1 ms-sm-2 me-1"
				:class="prevLink ? '' : 'disabled'" title="View previous message">
				<i class="bi bi-caret-left-fill"></i>
			</RouterLink>
			<RouterLink :to="'/view/' + nextLink" class="btn btn-outline-light" :class="nextLink ? '' : 'disabled'">
				<i class="bi bi-caret-right-fill" title="View next message"></i>
			</RouterLink>
		</div>
	</div>

	<div class="row flex-fill" style="min-height:0">
		<div class="d-none d-md-block col-xl-2 col-md-3 mh-100 position-relative"
			style="overflow-y: auto; overflow-x: hidden;">

			<div class="list-group my-2">
				<button @click="goBack()" class="list-group-item list-group-item-action">
					<i class="bi bi-arrow-return-left me-1"></i>
					<span class="ms-1">Return</span>
					<span class="badge rounded-pill ms-1 float-end text-bg-secondary" title="Unread messages"
						v-if="mailbox.unread && !errorMessage">
						{{ formatNumber(mailbox.unread) }}
					</span>
				</button>
			</div>

			<div class="card mt-4" v-if="!errorMessage">
				<div class="card-body text-body-secondary small">
					<p class="card-text">
						<b>Message date:</b><br>
						<small>{{ messageDate(message.Date) }}</small>
					</p>
					<p class="card-text">
						<b>Size:</b> {{ getFileSize(message.Size) }}
					</p>
					<p class="card-text" v-if="allAttachments(message).length">
						<b>Attachments:</b> {{ allAttachments(message).length }}
					</p>
				</div>
			</div>
			<AboutMailpit />
		</div>

		<div class="col-xl-10 col-md-9 mh-100 ps-0 ps-md-2 pe-0">
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
