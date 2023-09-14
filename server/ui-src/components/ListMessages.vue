<script>
import CommonMixins from '../mixins/CommonMixins.js'
import { mailbox } from '../stores/mailbox.js'
import moment from 'moment'

export default {
	mixins: [
		CommonMixins
	],

	data() {
		return {
			mailbox,
		}
	},

	mounted() {
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
	},

	methods: {
		getRelativeCreated: function (message) {
			let d = new Date(message.Created)
			return moment(d).fromNow().toString()
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
	}
}
</script>

<template>
	<template v-if="mailbox.messages && mailbox.messages.length">
		<div class="list-group my-2">
			<RouterLink v-for="message in mailbox.messages" :to="'/view/' + message.ID" :key="message.ID"
				class="row gx-1 message d-flex small list-group-item list-group-item-action border-start-0 border-end-0"
				:class="message.Read ? 'read' : '', isSelected(message.ID) ? 'selected' : ''">
				<!-- <a v-for="message in messages" :href="'#' + message.ID" :key="message.ID"
				Av-on:click.ctrl="toggleSelected($event, message.ID)" Av-on:click.shift="selectRange($event, message.ID)"
				class="row gx-1 message d-flex small list-group-item list-group-item-action border-start-0 border-end-0"
				:class="message.Read ? 'read' : '', isSelected(message.ID) ? 'selected' : ''"> -->
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
				<!-- </a> -->
			</RouterLink>
		</div>
	</template>
	<template v-else>
		<p class="text-center mt-5">There are no messages</p>
	</template>
</template>
