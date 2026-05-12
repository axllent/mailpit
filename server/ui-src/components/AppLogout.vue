<script>
import { logout, oidcEnabled } from "../services/oidcAuth";

export default {
	computed: {
		show() {
			return oidcEnabled();
		},
	},
	methods: {
		signOut() {
			logout();
		},
	},
};
</script>

<template>
	<button
		v-if="show"
		type="button"
		class="btn btn-sm btn-outline-light position-fixed top-0 end-0 mt-2 me-2 d-print-none"
		style="z-index: 1050"
		title="Sign out"
		@click="signOut"
	>
		<i class="bi bi-box-arrow-right"></i>
		<span class="d-none d-md-inline ms-1">Sign out</span>
	</button>
</template>
