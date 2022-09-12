
<script>
import commonMixins from '../mixins.js';

export default {
	props: {
		message: Object,
		attachments: Object
	},

	mixins: [commonMixins]
}
</script>

<template>
	<div class="mt-4 border-top pt-4">
		<a v-for="part in attachments" :href="'api/'+message.ID+'/part/'+part.PartID" class="card attachment float-start me-3 mb-3" target="_blank" style="width: 180px">
			<img v-if="isImage(part)" :src="'api/'+message.ID+'/part/'+part.PartID+'/thumb'" class="card-img-top" alt="">
			<img v-else src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAALQAAAB4AQMAAABhKUq+AAAAA1BMVEX///+nxBvIAAAAGUlEQVQYGe3BgQAAAADDoPtTT+EA1QAAgFsLQAAB12s2WgAAAABJRU5ErkJggg==" class="card-img-top" alt="">
			<div class="icon" v-if="!isImage(part)">
				<i class="bi" :class="attachmentIcon(part)"></i>
			</div>
			<div class="card-body border-0">
				<p class="mb-1 text-muted">
					<i class="bi me-1" :class="attachmentIcon(part)"></i>
					<small>{{ getFileSize(part.Size) }}</small>
				</p>
				<p class="card-text mb-0 small">
					{{ part.FileName != '' ? part.FileName : '[ unknown ]' }}
				</p>
			</div>
				<div class="card-footer small border-0 text-center text-truncate">
				{{ part.FileName != '' ? part.FileName : '[ unknown ]' }}
			</div>
		</a>
	</div>
</template>
