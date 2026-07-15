<script>
import CommonMixins from "../mixins/CommonMixins";
import { mailbox } from "../stores/mailbox";
import { pagination } from "../stores/pagination";

export default {
	mixins: [CommonMixins],

	data() {
		return {
			mailbox,
			pagination,
		};
	},

	computed: {
		// the username mailbox currently being viewed, or "" for All mail
		current() {
			return this.$route.name === "mailbox" ? this.$route.params.username : "";
		},
	},

	methods: {
		isActive(username) {
			return this.current === username;
		},

		selectMailbox() {
			// switching mailbox starts a fresh view (clears any within-mailbox filter)
			pagination.start = 0;
			this.hideNav();
		},
	},
};
</script>

<template>
	<template v-if="mailbox.usernames && mailbox.usernames.length">
		<div class="mt-4 text-muted">
			<small class="text-uppercase">Mailbox</small>
		</div>
		<div class="dropdown mt-1 mb-2">
			<button
				class="btn btn-outline-secondary btn-sm dropdown-toggle w-100 d-flex justify-content-between align-items-center"
				type="button"
				data-bs-toggle="dropdown"
				aria-expanded="false"
			>
				<span class="text-truncate">
					<i class="bi bi-inbox-fill me-1"></i>
					<template v-if="current">{{ current }}</template>
					<template v-else>All mail</template>
				</span>
			</button>
			<ul class="dropdown-menu w-100">
				<li>
					<RouterLink to="/" class="dropdown-item" :class="!current ? 'active' : ''" @click="selectMailbox">
						<i class="bi bi-inboxes-fill me-1"></i>
						All mail
					</RouterLink>
				</li>
				<li><hr class="dropdown-divider" /></li>
				<li v-for="username in mailbox.usernames" :key="username">
					<RouterLink
						:to="'/mailbox/' + encodeURIComponent(username)"
						class="dropdown-item"
						:class="isActive(username) ? 'active' : ''"
						@click="selectMailbox"
					>
						<i class="bi bi-inbox me-1"></i>
						{{ username }}
					</RouterLink>
				</li>
			</ul>
		</div>
	</template>
</template>
