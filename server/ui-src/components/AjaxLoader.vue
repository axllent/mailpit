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
	<div v-if="loading > 0" class="loader">
		<div class="d-flex justify-content-center align-items-center h-100">
			<div class="spinner-border text-muted" role="status">
				<span class="visually-hidden">Loading...</span>
			</div>
		</div>
	</div>
</template>
