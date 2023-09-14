import CommonMixins from './CommonMixins.js'
import { mailbox } from "../stores/mailbox.js"
import { pagination } from "../stores/pagination.js"

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
			pagination.start = 0;
			this.loadMessages()
		},

		loadMessages: function () {
			if (!this.apiURI) {
				alert('apiURL not set!')
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

				// pagination.total = response.data.messages_count
				// self.existingTags = JSON.parse(JSON.stringify(self.tags))

				// if pagination > 0 && results == 0 reload first page (prune)
				if (response.data.count == 0 && response.data.start > 0) {
					pagination.start = 0
					return self.loadMessages()
				}

				if (!window.scrollInPlace) {
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
