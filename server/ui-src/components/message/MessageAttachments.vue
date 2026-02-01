<script>
import commonMixins from "../../mixins/CommonMixins";
import { mailbox } from "../../stores/mailbox";
import ICAL from "ical.js";
import dayjs from "dayjs";

export default {
	mixins: [commonMixins],

	props: {
		message: {
			type: Object,
			required: true,
		},
		attachments: {
			type: Object,
			required: true,
		},
	},

	data() {
		return {
			mailbox,
			ical: false,
		};
	},

	methods: {
		openAttachment(part, e) {
			const filename = part.FileName;
			const contentType = part.ContentType;
			const href = this.resolve("/api/v1/message/" + this.message.ID + "/part/" + part.PartID);
			if (filename.match(/\.ics$/i) || contentType === "text/calendar") {
				e.preventDefault();

				this.get(href, null, (response) => {
					const comp = new ICAL.Component(ICAL.parse(response.data));
					const vevent = comp.getFirstSubcomponent("vevent");
					if (!vevent) {
						alert("Error parsing ICS file");
						return;
					}
					const event = new ICAL.Event(vevent);

					const summary = {};
					summary.link = href;
					summary.status = vevent.getFirstPropertyValue("status");
					summary.url = vevent.getFirstPropertyValue("url");
					summary.summary = event.summary;
					summary.description = event.description;
					summary.location = event.location;
					summary.start = dayjs(event.startDate).format("ddd, D MMM YYYY, h:mm a");
					summary.end = dayjs(event.endDate).format("ddd, D MMM YYYY, h:mm a");
					summary.isRecurring = event.isRecurring();
					summary.organizer = event.organizer ? event.organizer.replace(/^mailto:/, "") : false;
					summary.attendees = [];
					event.attendees.forEach((a) => {
						if (a.jCal[1].cn) {
							summary.attendees.push(a.jCal[1].cn);
						}
					});

					comp.getAllSubcomponents("vtimezone").forEach((vtimezone) => {
						summary.timezone = vtimezone.getFirstPropertyValue("tzid");
					});

					this.ical = summary;

					// display modal
					this.modal("ICSView").show();
				});
			}
		},
	},
};
</script>

<template>
	<hr />

	<button
		class="btn btn-sm btn-outline-secondary mb-3"
		@click="mailbox.showAttachmentDetails = !mailbox.showAttachmentDetails"
	>
		<i class="bi me-1" :class="mailbox.showAttachmentDetails ? 'bi-eye-slash' : 'bi-eye'"></i>
		{{ mailbox.showAttachmentDetails ? "Hide" : "Show" }} attachment details
	</button>

	<div class="row gx-1 w-100">
		<div
			v-for="part in attachments"
			:key="part.PartID"
			:class="mailbox.showAttachmentDetails ? 'col-12' : 'col-auto'"
		>
			<div class="row gx-1 mb-3">
				<div class="col-auto">
					<a
						:href="resolve('/api/v1/message/' + message.ID + '/part/' + part.PartID)"
						class="card attachment float-start me-3 mb-3"
						target="_blank"
						style="width: 180px"
						@click="openAttachment(part, $event)"
					>
						<img
							v-if="isImage(part)"
							:src="resolve('/api/v1/message/' + message.ID + '/part/' + part.PartID + '/thumb')"
							class="card-img-top"
							alt=""
						/>
						<img
							v-else
							src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAALQAAAB4AQMAAABhKUq+AAAAA1BMVEX///+nxBvIAAAAGUlEQVQYGe3BgQAAAADDoPtTT+EA1QAAgFsLQAAB12s2WgAAAABJRU5ErkJggg=="
							class="card-img-top"
							alt=""
						/>
						<div v-if="!isImage(part)" class="icon">
							<i class="bi" :class="attachmentIcon(part)"></i>
						</div>
						<div class="card-body border-0">
							<p class="mb-1">
								<i class="bi me-1" :class="attachmentIcon(part)"></i>
								<small>{{ getFileSize(part.Size) }}</small>
							</p>
							<p class="card-text mb-0 small">
								{{ part.FileName != "" ? part.FileName : "[ unknown ]" + part.ContentType }}
							</p>
						</div>
						<div class="card-footer small border-0 text-center text-truncate">
							{{ part.FileName != "" ? part.FileName : "[ unknown ]" + part.ContentType }}
						</div>
					</a>
				</div>
				<div v-if="mailbox.showAttachmentDetails" class="col">
					<h5 class="mb-1">
						<a
							:href="resolve('/api/v1/message/' + message.ID + '/part/' + part.PartID)"
							class="me-2"
							@click="openAttachment(part, $event)"
						>
							{{ part.FileName != "" ? part.FileName : "[ unknown ]" + part.ContentType }}
						</a>
						<small class="text-muted fw-light">
							<small>({{ getFileSize(part.Size) }})</small>
						</small>
					</h5>
					<p class="mb-1 small"><strong>Disposition</strong>: {{ part.ContentDisposition }}</p>
					<p class="mb-2 small">
						<strong>Content type</strong>: <code>{{ part.ContentType }}</code>
					</p>
					<p class="m-0 small">
						<strong>MD5</strong>:
						<button
							class="btn btn-sm btn-link p-0"
							title="Click to copy to clipboard"
							@click="copyToClipboard(part.Checksums.MD5, $event)"
						>
							{{ part.Checksums.MD5 }}
							<i v-if="!copiedText[part.Checksums.MD5]" class="bi bi-clipboard ms-1"></i>
							<i v-else class="bi bi-check2-square ms-1 text-success"></i>
						</button>
					</p>
					<p class="m-0 small">
						<strong>SHA1</strong>:
						<button
							class="btn btn-link p-0"
							title="Click to copy to clipboard"
							@click="copyToClipboard(part.Checksums.SHA1, $event)"
						>
							{{ part.Checksums.SHA1 }}
							<i v-if="!copiedText[part.Checksums.SHA1]" class="bi bi-clipboard ms-1"></i>
							<i v-else class="bi bi-check2-square ms-1 text-success"></i>
						</button>
					</p>
					<p class="m-0 small">
						<strong>SHA256</strong>:
						<button
							class="btn btn-sm btn-link p-0"
							title="Click to copy to clipboard"
							@click="copyToClipboard(part.Checksums.SHA256, $event)"
						>
							{{ part.Checksums.SHA256 }}
							<i v-if="!copiedText[part.Checksums.SHA256]" class="bi bi-clipboard ms-1"></i>
							<i v-else class="bi bi-check2-square ms-1 text-success"></i>
						</button>
					</p>
				</div>
			</div>
		</div>
	</div>

	<!-- ICS Modal -->
	<div id="ICSView" class="modal fade" tabindex="-1" aria-hidden="true">
		<div class="modal-dialog modal-dialog-centered modal-dialog-scrollable modal-lg">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title fs-5">
						<i class="bi bi-calendar-event me-2"></i>
						iCalendar summary
					</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div v-if="ical" class="modal-body">
					<table class="table">
						<tbody>
							<tr v-if="ical.summary">
								<th>Summary</th>
								<td>{{ ical.summary }}</td>
							</tr>
							<tr v-if="ical.description">
								<th>Description</th>
								<td>{{ ical.description }}</td>
							</tr>
							<tr>
								<th>When</th>
								<td>
									{{ ical.start }} &mdash; {{ ical.end }}
									<span v-if="ical.isRecurring">(recurring)</span>
								</td>
							</tr>
							<tr v-if="ical.status">
								<th>Status</th>
								<td>{{ ical.status }}</td>
							</tr>
							<tr v-if="ical.location">
								<th>Location</th>
								<td>{{ ical.location }}</td>
							</tr>
							<tr v-if="ical.url">
								<th>URL</th>
								<td>
									<a :href="ical.url" target="_blank">{{ ical.url }}</a>
								</td>
							</tr>
							<tr v-if="ical.organizer">
								<th>Organizer</th>
								<td>{{ ical.organizer }}</td>
							</tr>
							<tr v-if="ical.attendees.length">
								<th>Attendees</th>
								<td>
									<span v-for="(a, i) in ical.attendees" :key="'attendee_' + i">
										<template v-if="i > 0">,</template>
										{{ a }}
									</span>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
					<a class="btn btn-primary" target="_blank" :href="ical.link"> Download attachment </a>
				</div>
			</div>
		</div>
	</div>
</template>
