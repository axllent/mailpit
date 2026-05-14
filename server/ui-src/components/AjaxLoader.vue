<script>
export default {
	props: {
		loading: {
			type: Number,
			default: 0,
		},
	},
};
</script>

<template>
	<div v-if="loading > 0" class="loader" role="status" aria-live="polite" aria-label="Loading">
		<div class="loader-bar"></div>
	</div>
</template>
