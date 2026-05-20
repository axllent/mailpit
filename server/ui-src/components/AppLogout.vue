<script>
import { getUser, logout, oidcEnabled } from "../services/oidcAuth";

export default {
	data() {
		return {
			displayName: "",
		};
	},
	computed: {
		show() {
			return oidcEnabled();
		},
	},
	async mounted() {
		if (!this.show) return;
		const u = await getUser();
		if (!u || !u.profile) return;
		this.displayName = u.profile.name || u.profile.preferred_username || u.profile.email || "Signed in";
	},
	methods: {
		signOut() {
			logout();
		},
	},
};
</script>

<template>
	<div v-if="show" class="bg-body ms-sm-n1 me-sm-n1 pb-2 text-muted small">
		<button type="button" class="text-muted btn btn-sm" :title="displayName">
			<i class="bi bi-person-fill me-1"></i>
			{{ displayName || "Signed in" }}
		</button>
		<button type="button" class="btn btn-sm btn-outline-secondary float-end" title="Sign out" @click="signOut">
			<i class="bi bi-box-arrow-right"></i>
		</button>
	</div>
</template>
