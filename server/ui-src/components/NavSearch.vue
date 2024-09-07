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

			const uri = this.resolve(`/api/v1/search`) + '?query=' + encodeURIComponent(s)
			this.delete(uri, false, (response) => {
				this.$router.push('/')
			})
		}
	}
}
</script>

<template>
	<template v-if="!modals">
		<div class="text-center badge text-bg-primary py-2 my-2 w-100" v-if="mailbox.uiConfig.Label">
			<div class="text-truncate fw-normal">
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
				<button class="list-group-item list-group-item-action" data-bs-toggle="modal"
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
