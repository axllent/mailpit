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
		};
	},

	mounted() {
		const uri = this.resolve("/api/v1/message/" + this.message.ID + "/headers");
		this.get(uri, false, (response) => {
			this.headers = response.data;
		});
	},
};
</script>

<template>
	<div v-if="headers" class="small">
		<div v-for="(values, k) in headers" :key="'headers_' + k" class="row mb-2 pb-2 border-bottom w-100">
			<div class="col-md-4 col-lg-3 col-xl-2 mb-2">
				<b>{{ k }}</b>
			</div>
			<div class="col-md-8 col-lg-9 col-xl-10 text-body-secondary">
				<div v-for="(x, i) in values" :key="'line_' + i" class="mb-2 text-break">{{ x }}</div>
			</div>
		</div>
	</div>
</template>
