<script>
export default {
	props: {
		loading: Number,
	},
}
</script>

<template>
	<div class="loader" v-if="loading > 0">
		<div class="d-flex justify-content-center align-items-center h-100">
			<div class="spinner-border text-secondary" role="status">
				<span class="visually-hidden">Loading...</span>
			</div>
		</div>
	</div>
</template>
