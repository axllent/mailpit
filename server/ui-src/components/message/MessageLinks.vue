<script>
import axios from "axios";
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
			error: false,
			links: false,
			loaded: false,
			loading: false,
		};
	},

	mounted() {
		this.loadLinks();
	},

	methods: {
		copyToClipboard(text) {
			navigator.clipboard.writeText(text);
		},

		loadLinks() {
			this.loading = true;
			const uri = this.resolve("/api/v1/message/" + this.message.ID + "/links");

			axios
				.get(uri, null)
				.then((result) => {
					this.links = result.data;
					this.error = false;
					this.loaded = true;
				})
				.catch((error) => {
					if (error.response && error.response.data) {
						if (error.response.data.Error) {
							this.error = error.response.data.Error;
						} else {
							this.error = error.response.data;
						}
					} else if (error.request) {
						this.error = "Error sending data to the server. Please try again.";
					} else {
						this.error = error.message;
					}
				})
				.then(() => {
					this.loading = false;
				});
		},
	},
};
</script>

<template>
	<div class="pe-3">
		<div class="row mb-3 align-items-center">
			<div class="col">
				<h4 class="mb-0">
					<template v-if="loading">
						Loading links...
						<div class="ms-1 spinner-border spinner-border-sm" role="status">
							<span class="visually-hidden">Loading...</span>
						</div>
					</template>
					<template v-else-if="links">
						<template v-if="links.Total > 0">
							{{ formatNumber(links.Total) }} link<template v-if="links.Total != 1">s</template>
						</template>
						<template v-else> No links detected </template>
					</template>
					<template v-else> Links </template>
				</h4>
			</div>
		</div>

		<template v-if="error">
			<p>Failed to load links:</p>
			<div class="alert alert-warning">
				{{ error }}
			</div>
		</template>

		<template v-else-if="links && links.Total > 0">
			<div class="card">
				<ul class="list-group list-group-flush">
					<li v-for="(url, i) in links.Links" :key="'link' + i" class="list-group-item">
						<a :href="url" target="_blank" class="no-icon">{{ url }}</a>
						<button
							class="btn btn-sm btn-link p-0 ms-2"
							title="Copy link"
							@click="copyToClipboard(url)"
						>
							<i class="bi bi-clipboard"></i>
						</button>
					</li>
				</ul>
			</div>
		</template>

		<template v-else-if="links && links.Total === 0">
			<p class="text-muted">No links were found in this message.</p>
		</template>
	</div>
</template>
