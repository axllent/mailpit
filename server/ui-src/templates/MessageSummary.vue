<script>
import commonMixins from '../mixins.js';

export default {
	props: {
		message: Object
	},

	mixins: [commonMixins]
}
</script>

<template>
	<div class="card mt-4">
		<div class="card-body text-muted small">
			<p class="card-text">
				<b>Message date:</b><br>
				<small>{{ messageDate(message.Date) }}</small>
			</p>
			<p class="card-text">
				<b>Size:</b> {{ getFileSize(message.Size) }}
			</p>
			<p class="card-text" v-if="allAttachments(message).length">
				<b>Attachments:</b> {{ allAttachments(message).length }}
			</p>
		</div>
	</div>
</template>
