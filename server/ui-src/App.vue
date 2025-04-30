<script>
import CommonMixins from './mixins/CommonMixins'
import Favicon from './components/Favicon.vue'
import AppBadge from './components/AppBadge.vue'
import Notifications from './components/Notifications.vue'
import EditTags from './components/EditTags.vue'
import { mailbox } from "./stores/mailbox"

export default {
	mixins: [CommonMixins],

	components: {
		Favicon,
		AppBadge,
		Notifications,
		EditTags
	},

	beforeMount() {
		// load global config
		this.get(this.resolve('/api/v1/webui'), false, function (response) {
			mailbox.uiConfig = response.data

			if (mailbox.uiConfig.Label) {
				document.title = document.title + ' - ' + mailbox.uiConfig.Label
			} else {
				document.title = document.title + ' - ' + location.hostname
			}
		})
	},

	watch: {
		$route(to, from) {
			// hide mobile menu on URL change
			this.hideNav()
		}
	},

}
</script>

<template>
	<RouterView />
	<Favicon />
	<AppBadge />
	<Notifications />
	<EditTags />
</template>
