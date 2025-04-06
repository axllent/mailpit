<script>
import NavSelected from '../components/NavSelected.vue'
import AjaxLoader from './AjaxLoader.vue'
import CommonMixins from '../mixins/CommonMixins'
import { mailbox } from '../stores/mailbox'
import { pagination } from '../stores/pagination'

export default {
	mixins: [CommonMixins],

	components: {
		NavSelected,
		AjaxLoader,
	},

	props: {
		modals: {
			type: Boolean,
			default: false,
		}
	},

	emits: ['loadMessages'],

	data() {
		return {
			mailbox,
			pagination,
		}
	},

	methods: {
		loadMessages() {
			this.hideNav() // hide mobile menu
			this.$emit('loadMessages')
		},

		deleteAllMessages() {
			const s = this.getSearch()
			if (!s) {
				return
			}

			let uri = this.resolve(`/api/v1/search`) + '?query=' + encodeURIComponent(s)
			if (mailbox.timeZone != '' && (s.indexOf('after:') != -1 || s.indexOf('before:') != -1)) {
				uri += '&tz=' + encodeURIComponent(mailbox.timeZone)
			}

			this.delete(uri, false, () => {
				this.$router.push('/')
			})
		},

		markAllRead() {
			const s = this.getSearch()
			if (!s) {
				return
			}

			let uri = this.resolve(`/api/v1/messages`)
			if (mailbox.timeZone != '' && (s.indexOf('after:') != -1 || s.indexOf('before:') != -1)) {
				uri += '?tz=' + encodeURIComponent(mailbox.timeZone)
			}

			this.put(uri, { 'read': true, "search": s }, () => {
				window.scrollInPlace = true
				this.loadMessages()
			})
		},
	}
}
</script>

<template>
	<template v-if="!modals">
		<div class="text-center badge text-bg-primary py-2 my-2 w-100" v-if="mailbox.uiConfig.Label">
			<div class="text-truncate fw-normal" style="line-height: 1rem">
				{{ mailbox.uiConfig.Label }}
			</div>
		</div>

		<div class="list-group my-2" :class="mailbox.uiConfig.Label ? 'mt-0' : ''">
			<RouterLink to="/" class="list-group-item list-group-item-action" @click="pagination.start = 0">
				<i class="bi bi-arrow-return-left me-1"></i>
				<span class="ms-1">Inbox</span>
				<span class="badge rounded-pill ms-1 float-end text-bg-secondary" title="Unread messages"
					v-if="mailbox.unread">
					{{ formatNumber(mailbox.unread) }}
				</span>
			</RouterLink>
			<template v-if="!mailbox.selected.length">
				<button v-if="mailbox.skipConfirmations" class="list-group-item list-group-item-action"
					:disabled="!mailbox.messages_unread" @click="markAllRead">
					<i class="bi bi-eye-fill me-1"></i>
					Mark all read
				</button>
				<button v-else class="list-group-item list-group-item-action" data-bs-toggle="modal"
					data-bs-target="#MarkAllReadModal" :disabled="!mailbox.messages_unread">
					<i class="bi bi-eye-fill me-1"></i>
					Mark all read
				</button>
				<button v-if="mailbox.skipConfirmations" class="list-group-item list-group-item-action"
					@click="deleteAllMessages" :disabled="!mailbox.count">
					<i class="bi bi-trash-fill me-1 text-danger"></i>
					Delete all
				</button>
				<button v-else class="list-group-item list-group-item-action" data-bs-toggle="modal"
					data-bs-target="#DeleteAllModal" :disabled="!mailbox.count">
					<i class="bi bi-trash-fill me-1 text-danger"></i>
					Delete all
				</button>
			</template>

			<NavSelected @loadMessages="loadMessages" />
		</div>
	</template>

	<template v-else>
		<!-- Modals -->
		<div class="modal fade" id="MarkAllReadModal" tabindex="-1" aria-labelledby="MarkAllReadModalLabel"
			aria-hidden="true">
			<div class="modal-dialog">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title" id="MarkAllReadModalLabel">Mark all search results as read?</h5>
						<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
					</div>
					<div class="modal-body">
						This will mark {{ formatNumber(mailbox.messages_unread) }}
						message<span v-if="mailbox.messages_unread > 1">s</span>
						matching <code>{{ getSearch() }}</code>
						as read.
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
						<button type="button" class="btn btn-success" data-bs-dismiss="modal"
							v-on:click="markAllRead">Confirm</button>
					</div>
				</div>
			</div>
		</div>

		<div class="modal fade" id="DeleteAllModal" tabindex="-1" aria-labelledby="DeleteAllModalLabel"
			aria-hidden="true">
			<div class="modal-dialog">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title" id="DeleteAllModalLabel">Delete all messages matching search?</h5>
						<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
					</div>
					<div class="modal-body">
						This will permanently delete {{ formatNumber(mailbox.count) }}
						message<span v-if="mailbox.count > 1">s</span> matching
						<code>{{ getSearch() }}</code>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Cancel</button>
						<button type="button" class="btn btn-danger" data-bs-dismiss="modal"
							v-on:click="deleteAllMessages">Delete</button>
					</div>
				</div>
			</div>
		</div>
	</template>

	<AjaxLoader :loading="loading" />
</template>
