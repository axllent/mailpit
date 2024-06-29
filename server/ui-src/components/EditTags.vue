<script>
import CommonMixins from '../mixins/CommonMixins'
import { mailbox } from '../stores/mailbox'

export default {
	mixins: [CommonMixins],

	data() {
		return {
			mailbox,
			editableTags: [],
			validTagRe: new RegExp(/^([a-zA-Z0-9\-\ \_\.]){1,}$/),
			tagToDelete: false,
		}
	},

	watch: {
		'mailbox.tags': {
			handler(tags) {
				this.editableTags = []
				tags.forEach((t) => {
					this.editableTags.push({ before: t, after: t })
				})
			},
			deep: true
		}
	},

	methods: {
		validTag(t) {
			if (!t.after.match(/^([a-zA-Z0-9\-\ \_\.]){1,}$/)) {
				return false
			}

			const lower = t.after.toLowerCase()
			for (let x = 0; x < this.editableTags.length; x++) {
				if (this.editableTags[x].before != t.before && lower == this.editableTags[x].before.toLowerCase()) {
					return false
				}
			}

			return true
		},

		renameTag(t) {
			if (!this.validTag(t) || t.before == t.after) {
				return
			}

			this.put(this.resolve(`/api/v1/tags/` + encodeURI(t.before)), { Name: t.after }, () => {
				// the API triggers a reload via websockets
			})
		},

		deleteTag() {
			this.delete(this.resolve(`/api/v1/tags/` + encodeURI(this.tagToDelete.before)), null, () => {
				// the API triggers a reload via websockets
				this.tagToDelete = false
			})
		},

		resetTagEdit(t) {
			for (let x = 0; x < this.editableTags.length; x++) {
				if (this.editableTags[x].before != t.before && this.editableTags[x].before != this.editableTags[x].after) {
					this.editableTags[x].after = this.editableTags[x].before
				}
			}
		}
	}
}
</script>

<template>
	<div class="modal fade" id="EditTagsModal" tabindex="-1" aria-labelledby="EditTagsModalLabel" aria-hidden="true"
		data-bs-keyboard="false">
		<div class="modal-dialog modal-lg">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title" id="EditTagsModalLabel">Edit tags</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
				</div>
				<div class="modal-body">
					<p>
						Renaming a tag will update the tag for all messages. Deleting a tag will only delete the tag
						itself, and not any messages which had the tag.
					</p>
					<div class="mb-3" v-for="t in editableTags">
						<div class="input-group has-validation">
							<input type="text" class="form-control" :class="!validTag(t) ? 'is-invalid' : ''"
								v-model.trim="t.after" aria-describedby="inputGroupPrepend" required
								@keydown.enter="renameTag(t)" @keydown.esc="t.after = t.before"
								@focus="resetTagEdit(t)">
							<button v-if="t.before != t.after" class="btn btn-success"
								@click="renameTag(t)">Save</button>
							<template v-else>
								<button class="btn btn-outline-danger"
									:class="tagToDelete.before == t.before ? 'text-white btn-danger' : ''"
									@click="!tagToDelete ? tagToDelete = t : deleteTag()" @blur="tagToDelete = false">
									<template v-if="tagToDelete == t">
										Confirm?
									</template>
									<template v-else>
										Delete
									</template>
								</button>
							</template>
							<div class="invalid-feedback">
								Invalid tag name
							</div>
						</div>
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Close</button>
				</div>
			</div>
		</div>
	</div>
</template>
