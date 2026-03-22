<script>
import commonMixins from "../../mixins/CommonMixins";

export default {
	mixins: [commonMixins],

	props: {
		message: {
			type: Object,
			required: true,
		},
	},

	data() {
		return {
			headers: false,
			filter: "",
		};
	},

	computed: {
		filteredHeaders() {
			if (this.filter === "") {
				return this.headers;
			}
			const searchWords = this.filter
				.toLowerCase()
				.split(/\s+/)
				.filter((x) => x.length > 0);

			const filtered = {};
			for (const k in this.headers) {
				const values = this.headers[k];
				const kLower = k.toLowerCase();
				if (searchWords.every((w) => kLower.includes(w))) {
					filtered[k] = values;
				} else {
					const matchingValues = values.filter((v) => {
						const vLower = v.toLowerCase();
						return searchWords.every((w) => vLower.includes(w));
					});
					if (matchingValues.length > 0) {
						filtered[k] = matchingValues;
					}
				}
			}

			return filtered;
		},
	},

	mounted() {
		const uri = this.resolve("/api/v1/message/" + this.message.ID + "/headers");
		this.get(uri, false, (response) => {
			this.headers = response.data;
		});
	},

	methods: {
		highlight(text) {
			const escaped = text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
			if (!this.filter || this.filter.trim() === "") {
				return escaped;
			}
			const words = this.filter
				.trim()
				.split(/\s+/)
				.filter((w) => w.length > 0)
				.map((w) => w.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"));
			const regex = new RegExp(words.join("|"), "gi");
			return escaped.replace(regex, "<mark>$&</mark>");
		},
	},
};
</script>

<template v-if="headers">
	<div class="row w-100 mb-3">
		<div class="col col-md-10 col-lg-7">
			<input
				v-model.trim="filter"
				type="search"
				class="form-control mb-3"
				placeholder="Filter headers..."
				aria-label="Filter headers"
			/>
		</div>
	</div>
	<div v-if="Object.keys(filteredHeaders).length > 0" class="small">
		<div v-for="(values, k) in filteredHeaders" :key="'headers_' + k" class="row mb-2 pb-2 border-bottom w-100">
			<div class="col-md-4 col-lg-3 col-xl-2 mb-2">
				<!-- eslint-disable-next-line vue/no-v-html -->
				<b v-html="highlight(k)"></b>
			</div>
			<div class="col-md-8 col-lg-9 col-xl-10 text-body-secondary">
				<!-- eslint-disable-next-line vue/no-v-html -->
				<div v-for="(x, i) in values" :key="'line_' + i" class="mb-2 text-break" v-html="highlight(x)"></div>
			</div>
		</div>
	</div>
	<div v-else class="text-body-secondary">No matching headers found.</div>
</template>
