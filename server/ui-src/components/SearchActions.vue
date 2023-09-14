<script>
import { mailbox } from '../stores/mailbox.js'
import { pagination } from '../stores/pagination.js'
import CommonMixins from '../mixins/CommonMixins.js'

export default {
	mixins: [CommonMixins],

	emits: ['loadMessages'],

	data() {
		return {
			mailbox,
			pagination,
		}
	},

	methods: {

	}
}
</script>

<template>
	<div class="list-group my-2">
		<RouterLink to="/" class="list-group-item list-group-item-action">
			<i class="bi bi-arrow-return-left me-1"></i>
			<span class="ms-1">Inbox</span>
			<span class="badge rounded-pill ms-1 float-end text-bg-secondary" title="Unread messages" v-if="mailbox.unread">
				{{ formatNumber(mailbox.unread) }}
			</span>
		</RouterLink>

		<!-- <button class="list-group-item list-group-item-action" data-bs-toggle="modal" data-bs-target="#MarkAllReadModal"
			:disabled="!mailbox.unread">
			<i class="bi bi-eye-fill me-1"></i>
			Mark all read
		</button>

		<button class="list-group-item list-group-item-action" data-bs-toggle="modal" data-bs-target="#DeleteAllModal"
			:disabled="!mailbox.total">
			<i class="bi bi-trash-fill me-1 text-danger"></i>
			Delete all
		</button> -->

	</div>
</template>
