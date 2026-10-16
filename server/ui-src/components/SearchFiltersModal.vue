<script>
import CommonMixins from "../mixins/CommonMixins";

export default {
	mixins: [CommonMixins],

	inject: ["eventBus"],

	methods: {
		applyFilter(filter) {
			this.modal("SearchFiltersModal").hide();
			this.eventBus.emit("apply-search", filter);
		},
	},
};
</script>

<template>
	<div
		id="SearchFiltersModal"
		class="modal fade"
		tabindex="-1"
		aria-labelledby="SearchFiltersModalLabel"
		aria-hidden="true"
	>
		<div class="modal-dialog modal-lg modal-dialog-scrollable">
			<div class="modal-content">
				<div class="modal-header">
					<h5 id="SearchFiltersModalLabel" class="modal-title">
						<i class="bi bi-funnel-fill text-primary me-2"></i>Mailpit search filters
					</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div class="modal-body">
					<p class="text-muted small mb-3">
						Filter messages using prefixes and special terms. Click any example to try it immediately. Terms
						are case-insensitive (ASCII).
					</p>

					<!-- Header & Address filters -->
					<div class="card mb-3">
						<div class="card-header fw-semibold small text-uppercase bg-body-tertiary">
							<i class="bi bi-person-lines-fill me-1"></i> Addresses & headers
						</div>
						<div class="table-responsive">
							<table class="table table-sm table-hover align-middle mb-0">
								<thead>
									<tr class="text-muted small">
										<th style="width: 25%">Filter</th>
										<th style="width: 45%">Description</th>
										<th style="width: 30%">Example (click to try)</th>
									</tr>
								</thead>
								<tbody class="small">
									<tr>
										<td><code>to:&lt;term&gt;</code></td>
										<td>Matches "To" recipient header</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('to:&quot;john doe&quot;')"
											>
												to:"john doe"
											</span>
										</td>
									</tr>
									<tr>
										<td><code>from:&lt;term&gt;</code></td>
										<td>Matches "From" sender header</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('from:&quot;support@example.com&quot;')"
											>
												from:"support@example.com"
											</span>
										</td>
									</tr>
									<tr>
										<td><code>subject:&lt;term&gt;</code></td>
										<td>Matches email subject line</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('subject:invoice')"
											>
												subject:invoice
											</span>
										</td>
									</tr>
									<tr>
										<td><code>cc:&lt;term&gt;</code></td>
										<td>Matches "Cc" recipient header</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('cc:&quot;team@example.com&quot;')"
											>
												cc:"team@example.com"
											</span>
										</td>
									</tr>
									<tr>
										<td><code>bcc:&lt;term&gt;</code></td>
										<td>Matches "Bcc" recipient header</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('bcc:&quot;audit@example.com&quot;')"
											>
												bcc:"audit@example.com"
											</span>
										</td>
									</tr>
									<tr>
										<td><code>reply-to:&lt;term&gt;</code></td>
										<td>Matches "Reply-To" header</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('reply-to:&quot;noreply@example.com&quot;')"
											>
												reply-to:"noreply@example.com"
											</span>
										</td>
									</tr>
									<tr>
										<td><code>addressed:&lt;term&gt;</code></td>
										<td>Matches any of From, To, Cc, Bcc or Reply-To</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('addressed:&quot;john&quot;')"
											>
												addressed:"john"
											</span>
										</td>
									</tr>
									<tr>
										<td><code>message-id:&lt;id&gt;</code></td>
										<td>Matches Message-Id header</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('message-id:12345')"
											>
												message-id:12345
											</span>
										</td>
									</tr>
								</tbody>
							</table>
						</div>
					</div>

					<!-- Status & Content filters -->
					<div class="card mb-3">
						<div class="card-header fw-semibold small text-uppercase bg-body-tertiary">
							<i class="bi bi-check2-circle me-1"></i> Message status & content
						</div>
						<div class="table-responsive">
							<table class="table table-sm table-hover align-middle mb-0">
								<thead>
									<tr class="text-muted small">
										<th style="width: 25%">Filter</th>
										<th style="width: 45%">Description</th>
										<th style="width: 30%">Example (click to try)</th>
									</tr>
								</thead>
								<tbody class="small">
									<tr>
										<td><code>is:unread</code></td>
										<td>Messages with unread status</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('is:unread')"
											>
												is:unread
											</span>
										</td>
									</tr>
									<tr>
										<td><code>is:read</code></td>
										<td>Messages marked as read</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('is:read')"
											>
												is:read
											</span>
										</td>
									</tr>
									<tr>
										<td><code>has:attachment</code></td>
										<td>Messages containing attachments</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('has:attachment')"
											>
												has:attachment
											</span>
										</td>
									</tr>
									<tr>
										<td><code>has:inline</code></td>
										<td>Messages containing inline images</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('has:inline')"
											>
												has:inline
											</span>
										</td>
									</tr>
									<tr>
										<td><code>is:tagged</code></td>
										<td>Messages having at least one tag</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('is:tagged')"
											>
												is:tagged
											</span>
										</td>
									</tr>
									<tr>
										<td><code>tag:&lt;name&gt;</code></td>
										<td>Messages with a specific tag</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('tag:host-1')"
											>
												tag:host-1
											</span>
										</td>
									</tr>
								</tbody>
							</table>
						</div>
					</div>

					<!-- Dates & Sizes -->
					<div class="card mb-3">
						<div class="card-header fw-semibold small text-uppercase bg-body-tertiary">
							<i class="bi bi-calendar-event me-1"></i> Dates & sizes
						</div>
						<div class="table-responsive">
							<table class="table table-sm table-hover align-middle mb-0">
								<thead>
									<tr class="text-muted small">
										<th style="width: 25%">Filter</th>
										<th style="width: 45%">Description</th>
										<th style="width: 30%">Example (click to try)</th>
									</tr>
								</thead>
								<tbody class="small">
									<tr>
										<td><code>before:&lt;date&gt;</code></td>
										<td>Messages before date (format: YYYY/MM/DD)</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('before:2024/04/01')"
											>
												before:2024/04/01
											</span>
										</td>
									</tr>
									<tr>
										<td><code>after:&lt;date&gt;</code></td>
										<td>Messages after date (format: YYYY/MM/DD)</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('after:2024/04/01')"
											>
												after:2024/04/01
											</span>
										</td>
									</tr>
									<tr>
										<td><code>larger:&lt;size&gt;</code></td>
										<td>Messages larger than size (K, KB, M, MB)</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('larger:2M')"
											>
												larger:2M
											</span>
										</td>
									</tr>
									<tr>
										<td><code>smaller:&lt;size&gt;</code></td>
										<td>Messages smaller than size (K, KB, M, MB)</td>
										<td>
											<span
												class="badge bg-secondary-subtle text-body border search-filter-badge"
												role="button"
												title="Click to apply"
												@click="applyFilter('smaller:1.5MB')"
											>
												smaller:1.5MB
											</span>
										</td>
									</tr>
								</tbody>
							</table>
						</div>
					</div>

					<!-- Combining & Negation -->
					<div class="card mb-1">
						<div class="card-header fw-semibold small text-uppercase bg-body-tertiary">
							<i class="bi bi-sliders me-1"></i> Combining & exclusions
						</div>
						<div class="card-body small">
							<ul class="mb-0 ps-3">
								<li class="mb-2">
									<strong>Exclude terms:</strong> Prefix any filter or word with <code>!</code> or
									<code>-</code> (e.g.
									<span
										class="badge bg-secondary-subtle text-body border search-filter-badge"
										role="button"
										@click="applyFilter('!is:read')"
										>!is:read</span
									>,
									<span
										class="badge bg-secondary-subtle text-body border search-filter-badge"
										role="button"
										@click="applyFilter('!has:attachment')"
										>!has:attachment</span
									>,
									<span
										class="badge bg-secondary-subtle text-body border search-filter-badge"
										role="button"
										@click="applyFilter('!from:noreply')"
										>!from:noreply</span
									>).
								</li>
								<li class="mb-2">
									<strong>Exact phrases:</strong> Wrap multi-word phrases in quotes (e.g.
									<span
										class="badge bg-secondary-subtle text-body border search-filter-badge"
										role="button"
										@click="applyFilter('&quot;john doe&quot;')"
										>"john doe"</span
									>,
									<span
										class="badge bg-secondary-subtle text-body border search-filter-badge"
										role="button"
										@click="applyFilter('subject:&quot;monthly report&quot;')"
										>subject:"monthly report"</span
									>).
								</li>
								<li>
									<strong>Combine filters:</strong> Space-separated terms are combined (e.g.
									<span
										class="badge bg-secondary-subtle text-body border search-filter-badge"
										role="button"
										@click="applyFilter('subject:invoice !is:read has:attachment')"
									>
										subject:invoice !is:read has:attachment
									</span>
									).
								</li>
							</ul>
						</div>
					</div>
				</div>
				<div class="modal-footer d-flex justify-content-between">
					<a
						href="https://mailpit.axllent.org/docs/usage/search-filters/"
						target="_blank"
						rel="noopener noreferrer"
						class="btn btn-sm btn-link text-decoration-none"
					>
						Search documentation
					</a>
					<button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Close</button>
				</div>
			</div>
		</div>
	</div>
</template>
