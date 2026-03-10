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
			searchQuery: "",
			filterType: "all",
			// Standard email headers per RFC 5322 and common extensions
			standardHeaders: [
				"Accept-Language",
				"Authentication-Results",
				"Bcc",
				"Cc",
				"Comments",
				"Content-Description",
				"Content-Disposition",
				"Content-ID",
				"Content-Language",
				"Content-Length",
				"Content-Location",
				"Content-Transfer-Encoding",
				"Content-Type",
				"Date",
				"DKIM-Signature",
				"Disposition-Notification-Options",
				"Disposition-Notification-To",
				"Downgraded-Bcc",
				"Downgraded-Cc",
				"Downgraded-Disposition-Notification-To",
				"Downgraded-Final-Recipient",
				"Downgraded-From",
				"Downgraded-In-Reply-To",
				"Downgraded-Mail-From",
				"Downgraded-Message-Id",
				"Downgraded-Original-Recipient",
				"Downgraded-Rcpt-To",
				"Downgraded-References",
				"Downgraded-Reply-To",
				"Downgraded-Resent-Bcc",
				"Downgraded-Resent-Cc",
				"Downgraded-Resent-From",
				"Downgraded-Resent-Reply-To",
				"Downgraded-Resent-Sender",
				"Downgraded-Resent-To",
				"Downgraded-Return-Path",
				"Downgraded-Sender",
				"Downgraded-To",
				"Errors-To",
				"From",
				"In-Reply-To",
				"Keywords",
				"List-Archive",
				"List-Help",
				"List-ID",
				"List-Owner",
				"List-Post",
				"List-Subscribe",
				"List-Unsubscribe",
				"List-Unsubscribe-Post",
				"Message-ID",
				"MIME-Version",
				"Original-From",
				"Original-Message-ID",
				"Original-Recipient",
				"Original-Subject",
				"Precedence",
				"Received",
				"Received-SPF",
				"References",
				"Reply-To",
				"Require-Recipient-Valid-Since",
				"Resent-Bcc",
				"Resent-Cc",
				"Resent-Date",
				"Resent-From",
				"Resent-Message-ID",
				"Resent-Reply-To",
				"Resent-Sender",
				"Resent-To",
				"Return-Path",
				"Sender",
				"Subject",
				"To",
			],
		};
	},

	computed: {
		filteredHeaders() {
			if (!this.headers) return {};

			const result = {};
			const query = this.searchQuery.toLowerCase();

			for (const [key, values] of Object.entries(this.headers)) {
				// Check filter type
				const isStandard = this.isStandardHeader(key);
				if (this.filterType === "standard" && !isStandard) continue;
				if (this.filterType === "non-standard" && isStandard) continue;

				// Check search query
				if (query) {
					const keyMatches = key.toLowerCase().includes(query);
					const valueMatches = values.some((v) => v.toLowerCase().includes(query));
					if (!keyMatches && !valueMatches) continue;
				}

				result[key] = values;
			}

			return result;
		},

		hasResults() {
			return Object.keys(this.filteredHeaders).length > 0;
		},

		filterLabel() {
			switch (this.filterType) {
				case "standard":
					return "Standard Headers";
				case "non-standard":
					return "Non-Standard Headers";
				default:
					return "All Headers";
			}
		},
	},

	mounted() {
		const uri = this.resolve("/api/v1/message/" + this.message.ID + "/headers");
		this.get(uri, false, (response) => {
			this.headers = response.data;
		});
	},

	methods: {
		isStandardHeader(header) {
			// Check if header is in standard list (case-insensitive)
			const headerLower = header.toLowerCase();
			return this.standardHeaders.some((h) => h.toLowerCase() === headerLower);
		},

		setFilter(type) {
			this.filterType = type;
		},

		highlightText(text) {
			// Escape HTML first
			let html = text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");

			// IPv4 addresses
			html = html.replace(/\b(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\b/g, '<span class="header-highlight">$1</span>');

			// IPv6 addresses (simplified pattern for common formats)
			html = html.replace(
				/\b((?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,7}:|(?:[0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,5}(?::[0-9a-fA-F]{1,4}){1,2}|(?:[0-9a-fA-F]{1,4}:){1,4}(?::[0-9a-fA-F]{1,4}){1,3}|(?:[0-9a-fA-F]{1,4}:){1,3}(?::[0-9a-fA-F]{1,4}){1,4}|(?:[0-9a-fA-F]{1,4}:){1,2}(?::[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:(?::[0-9a-fA-F]{1,4}){1,6}|:(?::[0-9a-fA-F]{1,4}){1,7}|::)\b/g,
				'<span class="header-highlight">$1</span>',
			);

			// DKIM/SPF/DMARC keywords
			html = html.replace(/\b(dkim)\b/gi, '<span class="header-highlight">$1</span>');
			html = html.replace(/\b(spf)\b/gi, '<span class="header-highlight">$1</span>');
			html = html.replace(/\b(dmarc)\b/gi, '<span class="header-highlight">$1</span>');

			return html;
		},
	},
};
</script>

<template>
	<div>
		<div class="row mb-3 align-items-center">
			<div class="col-md-6 col-lg-4 mb-2 mb-md-0">
				<input
					v-model="searchQuery"
					type="text"
					class="form-control"
					placeholder="Search headers..."
					aria-label="Search headers"
				/>
			</div>
			<div class="col-md-6 col-lg-8 text-md-end">
				<div class="dropdown d-inline-block">
					<button
						class="btn btn-outline-primary dropdown-toggle"
						type="button"
						data-bs-toggle="dropdown"
						aria-expanded="false"
					>
						{{ filterLabel }}
					</button>
					<ul class="dropdown-menu dropdown-menu-end">
						<li>
							<button
								class="dropdown-item"
								:class="{ active: filterType === 'all' }"
								@click="setFilter('all')"
							>
								All Headers
							</button>
						</li>
						<li>
							<button
								class="dropdown-item"
								:class="{ active: filterType === 'standard' }"
								@click="setFilter('standard')"
							>
								Standard Headers
							</button>
						</li>
						<li>
							<button
								class="dropdown-item"
								:class="{ active: filterType === 'non-standard' }"
								@click="setFilter('non-standard')"
							>
								Non-Standard Headers
							</button>
						</li>
					</ul>
				</div>
			</div>
		</div>

		<div v-if="headers" class="small">
			<template v-if="hasResults">
				<table class="table table-sm table-hover mb-0">
					<tbody>
						<!-- eslint-disable vue/no-v-html -->
						<tr
							v-for="(values, k, index) in filteredHeaders"
							:key="'headers_' + k"
							:class="index % 2 === 0 ? 'header-row-alt' : ''"
						>
							<th
								class="text-nowrap align-top pe-4"
								style="width: 180px"
								v-html="highlightText(k)"
							></th>
							<td class="text-body-secondary text-break">
								<div
									v-for="(x, i) in values"
									:key="'line_' + i"
									:class="{ 'mb-1': i < values.length - 1 }"
									v-html="highlightText(x)"
								></div>
							</td>
						</tr>
						<!-- eslint-enable vue/no-v-html -->
					</tbody>
				</table>
			</template>
			<div v-else class="text-muted">No headers match your search criteria.</div>
		</div>
	</div>
</template>
