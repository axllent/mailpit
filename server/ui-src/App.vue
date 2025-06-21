<script>
import CommonMixins from "./mixins/CommonMixins";
import Favicon from "./components/AppFavicon.vue";
import AppBadge from "./components/AppBadge.vue";
import Notifications from "./components/AppNotifications.vue";
import EditTags from "./components/EditTags.vue";
import { mailbox } from "./stores/mailbox";

export default {
	components: {
		Favicon,
		AppBadge,
		Notifications,
		EditTags,
	},

	mixins: [CommonMixins],

	watch: {
		$route(to, from) {
			// hide mobile menu on URL change
			this.hideNav();
		},
	},

	beforeMount() {
		// load global config
		this.get(this.resolve("/api/v1/webui"), false, (response) => {
			mailbox.uiConfig = response.data;

			if (mailbox.uiConfig.Label) {
				document.title = document.title + " - " + mailbox.uiConfig.Label;
			} else {
				document.title = document.title + " - " + location.hostname;
			}
		});
	},
};
</script>

<template>
	<RouterView />
	<Favicon />
	<AppBadge />
	<Notifications />
	<EditTags />
</template>
