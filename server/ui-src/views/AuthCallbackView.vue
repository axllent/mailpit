<script>
import { handleCallback } from "../services/oidcAuth";

export default {
	data() {
		return { error: "" };
	},
	async mounted() {
		try {
			const returnTo = await handleCallback();
			// Hard navigation: re-boot the SPA so the router guard sees
			// the freshly stored user. router.replace can be canceled
			// during the initial route resolution and leave us stuck
			// here.
			window.location.replace(returnTo || "/");
		} catch (err) {
			this.error = err && err.message ? err.message : String(err);
		}
	},
};
</script>

<template>
	<div class="d-flex align-items-center justify-content-center h-100">
		<div v-if="!error" class="text-muted">Signing you in…</div>
		<div v-else class="alert alert-danger"><strong>Sign-in failed:</strong> {{ error }}</div>
	</div>
</template>
