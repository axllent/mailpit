<script>
import commonMixins from '../../mixins/CommonMixins'
import ICAL from "ical.js"
import dayjs from 'dayjs'

export default {
	props: {
		message: Object,
		attachments: Object
	},

	mixins: [commonMixins],

	data() {
		return {
			ical: false
		}
	},

	methods: {
		openAttachment: function (part, e) {
			let filename = part.FileName
			let contentType = part.ContentType
			let href = this.resolve('/api/v1/message/' + this.message.ID + '/part/' + part.PartID)
			if (filename.match(/\.ics$/i) || contentType == 'text/calendar') {
				e.preventDefault()

				this.get(href, null, (response) => {
					let comp = new ICAL.Component(ICAL.parse(response.data))
					let vevent = comp.getFirstSubcomponent('vevent')
					if (!vevent) {
						alert('Error parsing ICS file')
						return
					}
					let event = new ICAL.Event(vevent)

					let summary = {}
					summary.link = href
					summary.status = vevent.getFirstPropertyValue('status')
					summary.url = vevent.getFirstPropertyValue('url')
					summary.summary = event.summary
					summary.description = event.description
					summary.location = event.location
					summary.start = dayjs(event.startDate).format('ddd, D MMM YYYY, h:mm a')
					summary.end = dayjs(event.endDate).format('ddd, D MMM YYYY, h:mm a')
					summary.isRecurring = event.isRecurring()
					summary.organizer = event.organizer ? event.organizer.replace(/^mailto:/, '') : false
					summary.attendees = []
					event.attendees.forEach((a) => {
						if (a.jCal[1].cn) {
							summary.attendees.push(a.jCal[1].cn)
						}
					})

					comp.getAllSubcomponents("vtimezone").forEach((vtimezone) => {
						summary.timezone = vtimezone.getFirstPropertyValue("tzid")
					})

					this.ical = summary

					// display modal
					this.modal('ICSView').show()
				})
			}
		}
	},
}
</script>

<template>
	<div class="mt-4 border-top pt-4">
		<a v-for="part in attachments" :href="resolve('/api/v1/message/' + message.ID + '/part/' + part.PartID)"
			class="card attachment float-start me-3 mb-3" target="_blank" style="width: 180px"
			@click="openAttachment(part, $event)">
			<img v-if="isImage(part)"
				:src="resolve('/api/v1/message/' + message.ID + '/part/' + part.PartID + '/thumb')" class="card-img-top"
				alt="">
			<img v-else
				src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAALQAAAB4AQMAAABhKUq+AAAAA1BMVEX///+nxBvIAAAAGUlEQVQYGe3BgQAAAADDoPtTT+EA1QAAgFsLQAAB12s2WgAAAABJRU5ErkJggg=="
				class="card-img-top" alt="">
			<div class="icon" v-if="!isImage(part)">
				<i class="bi" :class="attachmentIcon(part)"></i>
			</div>
			<div class="card-body border-0">
				<p class="mb-1">
					<i class="bi me-1" :class="attachmentIcon(part)"></i>
					<small>{{ getFileSize(part.Size) }}</small>
				</p>
				<p class="card-text mb-0 small">
					{{ part.FileName != '' ? part.FileName : '[ unknown ]' + part.ContentType }}
				</p>
			</div>
			<div class="card-footer small border-0 text-center text-truncate">
				{{ part.FileName != '' ? part.FileName : '[ unknown ]' + part.ContentType }}
			</div>
		</a>
	</div>

	<div class="modal fade" id="ICSView" tabindex="-1" aria-hidden="true">
		<div class="modal-dialog modal-dialog-centered modal-dialog-scrollable modal-lg">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title fs-5">
						<i class="bi bi-calendar-event me-2"></i>
						iCalendar summary
					</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div class="modal-body" v-if="ical">
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
								<td> {{ ical.status }}</td>
							</tr>
							<tr v-if="ical.location">
								<th>Location</th>
								<td>{{ ical.location }}</td>
							</tr>
							<tr v-if="ical.url">
								<th>URL</th>
								<td><a :href="ical.url" target="_blank">{{ ical.url }}</a></td>
							</tr>
							<tr v-if="ical.organizer">
								<th>Organizer</th>
								<td>{{ ical.organizer }}</td>
							</tr>
							<tr v-if="ical.attendees.length">
								<th>Attendees</th>
								<td>
									<span v-for="(a, i) in ical.attendees">
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
					<a class="btn btn-primary" target="_blank" :href="ical.link">
						Download attachment
					</a>
				</div>
			</div>
		</div>
	</div>

</template>
