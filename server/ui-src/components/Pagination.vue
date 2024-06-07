<script>
import CommonMixins from '../mixins/CommonMixins'
import { mailbox } from '../stores/mailbox'
import { limitOptions, pagination } from '../stores/pagination'

export default {

	mixins: [CommonMixins],

	props: {
		total: Number,
	},

	data() {
		return {
			pagination,
			mailbox,
			limitOptions,
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
			this.updateQueryParams()
		},

		viewNext: function () {
			pagination.start = parseInt(pagination.start, 10) + parseInt(pagination.limit, 10)
			this.updateQueryParams()
		},

		viewPrev: function () {
			let s = pagination.start - pagination.limit
			if (s < 0) {
				s = 0
			}
			pagination.start = s
			this.updateQueryParams()
		},

		updateQueryParams: function () {
			const path = this.$route.path
			const p = {
				...this.$route.query
			}
			if (pagination.start > 0) {
				p.start = pagination.start.toString()
			} else {
				delete p.start
			}
			if (pagination.limit != pagination.defaultLimit) {
				p.limit = pagination.limit.toString()
			} else {
				delete p.limit
			}
			const params = new URLSearchParams(p)
			this.$router.push(path + '?' + params.toString())
		},
	}
}

</script>
<template>
	<select v-model="pagination.limit" @change="changeLimit" class="form-select form-select-sm d-inline w-auto me-2"
		:disabled="total == 0">
		<option v-for="option in limitOptions" :key="option" :value="option">{{ option }}</option>
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
