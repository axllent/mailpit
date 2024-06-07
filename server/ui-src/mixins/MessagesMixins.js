import CommonMixins from './CommonMixins.js'
import { mailbox } from '../stores/mailbox.js'
import { pagination } from '../stores/pagination.js'

export default {
	mixins: [CommonMixins],

	data() {
		return {
			apiURI: false,
			pagination,
			mailbox,
		}
	},

	watch: {
		'mailbox.refresh': function (v) {
			if (v) {
				// trigger a refresh
				this.loadMessages()
			}

			mailbox.refresh = false
		}
	},

	methods: {
		reloadMailbox: function () {
			pagination.start = 0
			this.loadMessages()
		},

		loadMessages: function () {
			if (!this.apiURI) {
				alert('apiURL not set!')
				return
			}

			// auto-pagination changes the URL but should not fetch new messages
			// when viewing page > 0 and new messages are received (inbox only)
			if (!mailbox.autoPaginating) {
				mailbox.autoPaginating = true // reset
				return
			}

			let self = this
			let params = {}
			mailbox.selected = []

			params['limit'] = pagination.limit
			if (pagination.start > 0) {
				params['start'] = pagination.start
			}

			self.get(this.apiURI, params, function (response) {
				mailbox.total = response.data.total // all messages
				mailbox.unread = response.data.unread // all unread messages
				mailbox.tags = response.data.tags // all tags
				mailbox.messages = response.data.messages // current messages
				mailbox.count = response.data.messages_count // total results for this mailbox/search
				// ensure the pagination remains consistent
				pagination.start = response.data.start

				if (response.data.count == 0 && response.data.start > 0) {
					pagination.start = 0
					return self.loadMessages()
				}

				if (mailbox.lastMessage) {
					window.setTimeout(() => {
						let m = document.getElementById(mailbox.lastMessage)
						if (m) {
							m.focus()
							// m.scrollIntoView({ behavior: 'smooth', block: 'center' })
							m.scrollIntoView({ block: 'center' })
						} else {
							let mp = document.getElementById('message-page')
							if (mp) {
								mp.scrollTop = 0
							}
						}

						mailbox.lastMessage = false
					}, 50)

				} else if (!window.scrollInPlace) {
					let mp = document.getElementById('message-page')
					if (mp) {
						mp.scrollTop = 0
					}
				}

				window.scrollInPlace = false
			})
		},
	}
}
