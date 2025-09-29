<script>
import CommonMixins from "../mixins/CommonMixins";
import { mailbox } from "../stores/mailbox";
import { limitOptions, pagination } from "../stores/pagination";

export default {
	mixins: [CommonMixins],

	props: {
		total: {
			type: Number,
			default: 0,
		},
	},

	data() {
		return {
			pagination,
			mailbox,
			limitOptions,
		};
	},

	computed: {
		canPrev() {
			return pagination.start > 0;
		},

		canNext() {
			return this.total > pagination.start + mailbox.messages.length;
		},

		// returns the number of next X messages
		nextMessages() {
			let t = pagination.start + parseInt(pagination.limit, 10);
			if (t > this.total) {
				t = this.total;
			}

			return t;
		},
	},

	methods: {
		changeLimit() {
			pagination.start = 0;
			this.updateQueryParams();
		},

		viewNext() {
			pagination.start = parseInt(pagination.start, 10) + parseInt(pagination.limit, 10);
			this.updateQueryParams();
		},

		viewPrev() {
			let s = pagination.start - pagination.limit;
			if (s < 0) {
				s = 0;
			}
			pagination.start = s;
			this.updateQueryParams();
		},

		updateQueryParams() {
			const path = this.$route.path;
			const p = {
				...this.$route.query,
			};
			if (pagination.start > 0) {
				p.start = pagination.start.toString();
			} else {
				delete p.start;
			}
			if (pagination.limit !== pagination.defaultLimit) {
				p.limit = pagination.limit.toString();
			} else {
				delete p.limit;
			}
			const params = new URLSearchParams(p);
			this.$router.push(path + "?" + params.toString());
		},
	},
};
</script>
<template>
	<select
		v-model="pagination.limit"
		class="form-select form-select-sm d-inline w-auto me-2 me-xl-3"
		:disabled="total == 0"
		@change="changeLimit"
	>
		<option v-for="option in limitOptions" :key="option" :value="option">{{ option }}</option>
	</select>

	<small>
		<template v-if="total > 0">
			{{ formatNumber(pagination.start + 1) }}-{{ formatNumber(nextMessages) }}
			<small>of</small>
			{{ formatNumber(total) }}
		</template>
		<span v-else class="text-light">0 of 0</span>
	</small>

	<button
		class="btn btn-outline-light ms-2 ms-xl-3 me-1"
		:disabled="!canPrev"
		:title="'View previous ' + pagination.limit + ' messages'"
		@click="viewPrev"
	>
		<i class="bi bi-caret-left-fill"></i>
	</button>
	<button
		class="btn btn-outline-light"
		:disabled="!canNext"
		:title="'View next ' + pagination.limit + ' messages'"
		@click="viewNext"
	>
		<i class="bi bi-caret-right-fill"></i>
	</button>
</template>
