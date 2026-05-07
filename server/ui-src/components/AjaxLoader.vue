<script>
export default {
	props: {
		loading: {
			type: Number,
			default: 0,
		},
	},

	data() {
		return {
			isVisible: false,
			showTimer: null,
		};
	},

	watch: {
		loading: {
			immediate: true,
			handler(v) {
				if (v > 0) {
					if (this.isVisible || this.showTimer) {
						return;
					}

					this.showTimer = window.setTimeout(() => {
						this.isVisible = this.loading > 0;
						this.showTimer = null;
					}, 200);
					return;
				}

				if (this.showTimer) {
					window.clearTimeout(this.showTimer);
					this.showTimer = null;
				}

				this.isVisible = false;
			},
		},
	},

	beforeUnmount() {
		if (this.showTimer) {
			window.clearTimeout(this.showTimer);
		}
	},
};
</script>

<template>
	<div v-if="isVisible" class="loader" role="status" aria-live="polite" aria-label="Loading">
		<div class="loader-bar"></div>
	</div>
</template>
