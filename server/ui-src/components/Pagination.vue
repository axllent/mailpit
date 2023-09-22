<script>
import CommonMixins from '../mixins/CommonMixins'
import { mailbox } from '../stores/mailbox'
import { pagination } from '../stores/pagination'

export default {

	mixins: [CommonMixins],

	props: {
		total: Number,
	},

	emits: ['loadMessages'],

	data() {
		return {
			pagination,
			mailbox,
		}
	},

	computed: {
		canPrev: function () {
			return pagination.start > 0
		},

		canNext: function () {
			return this.total > (pagination.start + mailbox.messages.length)
		},

		// returns the number of next X messages
		nextMessages: function () {
			let t = pagination.start + parseInt(pagination.limit, 10)
			if (t > this.total) {
				t = this.total
			}

			return t
		},
	},

	methods: {
		changeLimit: function () {
			pagination.start = 0
			this.$emit('loadMessages')
		},

		viewNext: function () {
			pagination.start = parseInt(pagination.start, 10) + parseInt(pagination.limit, 10)
			this.$emit('loadMessages')
		},

		viewPrev: function () {
			let s = pagination.start - pagination.limit
			if (s < 0) {
				s = 0
			}
			pagination.start = s
			this.$emit('loadMessages')
		},
	}
}

</script>
<template>
	<select v-model="pagination.limit" @change="changeLimit" class="form-select form-select-sm d-inline w-auto me-2"
		:disabled="total == 0">
		<option value="25">25</option>
		<option value="50">50</option>
		<option value="100">100</option>
		<option value="200">200</option>
	</select>

	<small>
		<template v-if="total > 0">
			{{ formatNumber(pagination.start + 1) }}-{{ formatNumber(nextMessages) }}
			<small>of</small>
			{{ formatNumber(total) }}
		</template>
		<span v-else class="text-muted">0 of 0</span>
	</small>

	<button class="btn btn-outline-light ms-2 me-1" :disabled="!canPrev" v-on:click="viewPrev"
		:title="'View previous ' + pagination.limit + ' messages'">
		<i class="bi bi-caret-left-fill"></i>
	</button>
	<button class="btn btn-outline-light" :disabled="!canNext" v-on:click="viewNext"
		:title="'View next ' + pagination.limit + ' messages'">
		<i class="bi bi-caret-right-fill"></i>
	</button>
</template>
