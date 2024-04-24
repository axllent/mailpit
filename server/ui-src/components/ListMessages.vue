<script>
import { mailbox } from '../stores/mailbox'
import CommonMixins from '../mixins/CommonMixins'
import dayjs from 'dayjs'

export default {
	mixins: [
		CommonMixins
	],

	props: {
		loadingMessages: Number, // use different name to `loading` as that is already in use in CommonMixins
	},

	data() {
		return {
			mailbox,
		}
	},

	mounted() {
		let relativeTime = require('dayjs/plugin/relativeTime')
		dayjs.extend(relativeTime)
		this.refreshUI()
	},

	methods: {
		refreshUI: function () {
			let self = this
			window.setTimeout(
				() => {
					self.$forceUpdate()
					self.refreshUI()
				},
				30000
			)
		},

		getRelativeCreated: function (message) {
			let d = new Date(message.Created)
			return dayjs(d).fromNow()
		},

		getPrimaryEmailTo: function (message) {
			for (let i in message.To) {
				return message.To[i].Address
			}

			return '[ Undisclosed recipients ]'
		},

		isSelected: function (id) {
			return mailbox.selected.indexOf(id) != -1
		},

		toggleSelected: function (e, id) {
			e.preventDefault()

			if (this.isSelected(id)) {
				mailbox.selected = mailbox.selected.filter(function (ele) {
					return ele != id
				})
			} else {
				mailbox.selected.push(id)
			}
		},

		selectRange: function (e, id) {
			e.preventDefault()

			let selecting = false
			let lastSelected = mailbox.selected.length > 0 && mailbox.selected[mailbox.selected.length - 1]
			if (lastSelected == id) {
				mailbox.selected = mailbox.selected.filter(function (ele) {
					return ele != id
				})
				return
			}

			if (lastSelected === false) {
				mailbox.selected.push(id)
				return
			}

			for (let d of mailbox.messages) {
				if (selecting) {
					if (!this.isSelected(d.ID)) {
						mailbox.selected.push(d.ID)
					}
					if (d.ID == lastSelected || d.ID == id) {
						// reached backwards select
						break
					}
				} else if (d.ID == id || d.ID == lastSelected) {
					if (!this.isSelected(d.ID)) {
						mailbox.selected.push(d.ID)
					}
					selecting = true
				}
			}
		},
	}
}
</script>

<template>
	<template v-if="mailbox.messages && mailbox.messages.length">
		<div class="list-group my-2">
			<RouterLink v-for="message in mailbox.messages" :to="'/view/' + message.ID" :key="message.ID"
				:id="message.ID"
				class="row gx-1 message d-flex small list-group-item list-group-item-action border-start-0 border-end-0"
				:class="message.Read ? 'read' : '', isSelected(message.ID) ? 'selected' : ''"
				v-on:click.ctrl="toggleSelected($event, message.ID)" v-on:click.shift="selectRange($event, message.ID)">
				<div class="col-lg-3">
					<div class="d-lg-none float-end text-muted text-nowrap small">
						<i class="bi bi-paperclip h6 me-1" v-if="message.Attachments"></i>
						{{ getRelativeCreated(message) }}
					</div>
					<div class="text-truncate d-lg-none privacy">
						<span v-if="message.From" :title="'From: ' + message.From.Address">{{
		message.From.Name ?
			message.From.Name : message.From.Address
	}}</span>
					</div>
					<div class="text-truncate d-none d-lg-block privacy">
						<b v-if="message.From" :title="'From: ' + message.From.Address">{{
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
					<div class="subject text-truncate text-spaces-nowrap">
						<b>{{ message.Subject != "" ? message.Subject : "[ no subject ]" }}</b>
					</div>
					<div v-if="message.Snippet != ''" class="small text-muted text-truncate">
						{{ message.Snippet }}
					</div>
					<div v-if="message.Tags.length">
						<RouterLink class="badge me-1" v-for="t in message.Tags" :to="'/search?q=' + tagEncodeURI(t)"
							:style="mailbox.showTagColors ? { backgroundColor: colorHash(t) } : { backgroundColor: '#6c757d' }"
							:title="'Filter messages tagged with ' + t">
							{{ t }}
						</RouterLink>
					</div>
				</div>
				<div class="d-none d-lg-block col-1 small text-end text-muted">
					<i class="bi bi-paperclip float-start h6" v-if="message.Attachments"></i>
					{{ getFileSize(message.Size) }}
				</div>
				<div class="d-none d-lg-block col-2 col-xxl-1 small text-end text-muted">
					{{ getRelativeCreated(message) }}
				</div>
			</RouterLink>
		</div>
	</template>
	<template v-else>
		<p class="text-center mt-5">
			<span v-if="loadingMessages > 0" class="text-secondary">
				Loading messages...
			</span>
			<template v-else-if="getSearch()">No results for <code>{{ getSearch() }}</code></template>
			<template v-else>No messages in your mailbox</template>
		</p>
	</template>
</template>
