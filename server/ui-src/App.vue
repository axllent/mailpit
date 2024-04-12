<script>
import CommonMixins from './mixins/CommonMixins'
import Favicon from './components/Favicon.vue'
import Notifications from './components/Notifications.vue'
import { RouterView } from 'vue-router'
import { mailbox } from "./stores/mailbox"

export default {
	mixins: [CommonMixins],

	components: {
		Favicon,
		Notifications,
	},

	beforeMount() {
		document.title = document.title + ' - ' + location.hostname

		// load global config
		this.get(this.resolve('/api/v1/webui'), false, function (response) {
			mailbox.uiConfig = response.data
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
	<Notifications />
</template>
