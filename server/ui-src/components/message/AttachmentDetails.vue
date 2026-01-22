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
			attachments: false,
			loaded: false,
			loading: false,
		};
	},

	computed: {
		totalAttachments() {
			if (!this.attachments) return 0;
			return this.attachments.Attachments.length + this.attachments.Inline.length;
		},
	},

	mounted() {
		this.loadAttachments();
	},

	methods: {
		loadAttachments() {
			this.loading = true;
			const uri = this.resolve("/api/v1/message/" + this.message.ID + "/attachments");

			axios
				.get(uri, null)
				.then((result) => {
					this.attachments = result.data;
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

		copyToClipboard(text) {
			navigator.clipboard.writeText(text);
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
						Loading attachments...
						<div class="ms-1 spinner-border spinner-border-sm" role="status">
							<span class="visually-hidden">Loading...</span>
						</div>
					</template>
					<template v-else-if="attachments">
						<template v-if="totalAttachments > 0">
							{{ formatNumber(totalAttachments) }} attachment<template v-if="totalAttachments != 1"
								>s</template
							>
						</template>
						<template v-else> No attachments </template>
					</template>
					<template v-else> Attachments </template>
				</h4>
			</div>
		</div>

		<template v-if="error">
			<p>Failed to load attachments:</p>
			<div class="alert alert-warning">
				{{ error }}
			</div>
		</template>

		<template v-else-if="attachments && totalAttachments > 0">
			<!-- Regular Attachments -->
			<template v-if="attachments.Attachments.length > 0">
				<h5 class="mb-3">Attachments</h5>
				<div v-for="(a, i) in attachments.Attachments" :key="'att' + i" class="card mb-3">
					<div class="card-header">
						<i :class="attachmentIcon(a)" class="me-2"></i>
						<strong>{{ a.FileName }}</strong>
						<a
							:href="resolve('/api/v1/message/' + message.ID + '/part/' + a.PartID)"
							class="btn btn-sm btn-outline-secondary float-end"
							target="_blank"
						>
							<i class="bi bi-download"></i>
							Download
						</a>
					</div>
					<div class="card-body">
						<table class="table table-sm table-borderless mb-0">
							<tbody>
								<tr>
									<th class="text-nowrap" style="width: 100px">File Size</th>
									<td>{{ getFileSize(a.Size) }} ({{ formatNumber(a.Size) }} bytes)</td>
								</tr>
								<tr>
									<th class="text-nowrap">File Type</th>
									<td>{{ a.ContentType }}</td>
								</tr>
								<tr>
									<th class="text-nowrap">MD5</th>
									<td>
										<code class="user-select-all">{{ a.MD5 }}</code>
										<button
											class="btn btn-sm btn-link p-0 ms-2"
											title="Copy MD5"
											@click="copyToClipboard(a.MD5)"
										>
											<i class="bi bi-clipboard"></i>
										</button>
									</td>
								</tr>
								<tr>
									<th class="text-nowrap">SHA1</th>
									<td>
										<code class="user-select-all">{{ a.SHA1 }}</code>
										<button
											class="btn btn-sm btn-link p-0 ms-2"
											title="Copy SHA1"
											@click="copyToClipboard(a.SHA1)"
										>
											<i class="bi bi-clipboard"></i>
										</button>
									</td>
								</tr>
								<tr>
									<th class="text-nowrap">SHA256</th>
									<td>
										<code class="user-select-all small">{{ a.SHA256 }}</code>
										<button
											class="btn btn-sm btn-link p-0 ms-2"
											title="Copy SHA256"
											@click="copyToClipboard(a.SHA256)"
										>
											<i class="bi bi-clipboard"></i>
										</button>
									</td>
								</tr>
							</tbody>
						</table>
					</div>
				</div>
			</template>

			<!-- Inline Attachments -->
			<template v-if="attachments.Inline.length > 0">
				<h5 class="mb-3 mt-4">Inline Attachments</h5>
				<div v-for="(a, i) in attachments.Inline" :key="'inline' + i" class="card mb-3">
					<div class="card-header">
						<i :class="attachmentIcon(a)" class="me-2"></i>
						<strong>{{ a.FileName }}</strong>
						<a
							:href="resolve('/api/v1/message/' + message.ID + '/part/' + a.PartID)"
							class="btn btn-sm btn-outline-secondary float-end"
							target="_blank"
						>
							<i class="bi bi-download"></i>
							Download
						</a>
					</div>
					<div class="card-body">
						<table class="table table-sm table-borderless mb-0">
							<tbody>
								<tr>
									<th class="text-nowrap" style="width: 100px">File Size</th>
									<td>{{ getFileSize(a.Size) }} ({{ formatNumber(a.Size) }} bytes)</td>
								</tr>
								<tr>
									<th class="text-nowrap">File Type</th>
									<td>{{ a.ContentType }}</td>
								</tr>
								<tr>
									<th class="text-nowrap">MD5</th>
									<td>
										<code class="user-select-all">{{ a.MD5 }}</code>
										<button
											class="btn btn-sm btn-link p-0 ms-2"
											title="Copy MD5"
											@click="copyToClipboard(a.MD5)"
										>
											<i class="bi bi-clipboard"></i>
										</button>
									</td>
								</tr>
								<tr>
									<th class="text-nowrap">SHA1</th>
									<td>
										<code class="user-select-all">{{ a.SHA1 }}</code>
										<button
											class="btn btn-sm btn-link p-0 ms-2"
											title="Copy SHA1"
											@click="copyToClipboard(a.SHA1)"
										>
											<i class="bi bi-clipboard"></i>
										</button>
									</td>
								</tr>
								<tr>
									<th class="text-nowrap">SHA256</th>
									<td>
										<code class="user-select-all small">{{ a.SHA256 }}</code>
										<button
											class="btn btn-sm btn-link p-0 ms-2"
											title="Copy SHA256"
											@click="copyToClipboard(a.SHA256)"
										>
											<i class="bi bi-clipboard"></i>
										</button>
									</td>
								</tr>
							</tbody>
						</table>
					</div>
				</div>
			</template>
		</template>

		<template v-else-if="attachments && totalAttachments === 0">
			<p class="text-muted">No attachments were found in this message.</p>
		</template>
	</div>
</template>
