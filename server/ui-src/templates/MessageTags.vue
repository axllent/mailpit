
<script>
import commonMixins from '../mixins.js';
import Tags from "bootstrap5-tags";

export default {
	props: {
		message: Object,
		existingTags: Array
	},

	mixins: [commonMixins],

	data() {
		return {
			messageTags: [],
		}
	},

	mounted() {
		let self = this;
		self.loaded = false;
		self.messageTags = self.message.Tags;
		// delay until vue has rendered
		self.$nextTick(function () {
			Tags.init("select[multiple]");
			self.$nextTick(function () {
				self.loaded = true;
			});
		});
	},

	watch: {
		messageTags() {
			if (this.loaded) {
				this.saveTags();
			}
		}
	},

	methods: {
		saveTags: function () {
			let self = this;

			var data = {
				ids: [this.message.ID],
				tags: this.messageTags
			}

			self.put('api/v1/tags', data, function (response) {
				self.scrollInPlace = true;
				self.$emit('loadMessages');
			});
		}
	}
}
</script>

<template>
	<tr class="small">
		<th>Tags</th>
		<td>
			<select class="form-select small tag-selector" v-model="messageTags" multiple data-allow-new="true"
				data-clear-end="true" data-allow-clear="true" data-placeholder="Add tags..."
				data-badge-style="secondary" data-regex="^([a-zA-Z0-9\-\ \_]){3,}$" data-separator="|,|">
				<option value="">Type a tag...</option><!-- you need at least one option with the placeholder -->
				<option v-for="t in existingTags" :value="t">{{ t }}</option>
			</select>
			<div class="invalid-feedback">Please select a valid tag.</div>
		</td>
	</tr>
</template>
