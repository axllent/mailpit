<script>
import CommonMixins from './mixins/CommonMixins'
import Notifications from './components/Notifications.vue'
import { RouterView } from 'vue-router'
import { mailbox } from "./stores/mailbox"

export default {
	mixins: [CommonMixins],

	components: {
		Notifications,
	},

	beforeMount() {
		document.title = document.title + ' - ' + location.hostname
		mailbox.showTagColors = localStorage.getItem('showTagsColors') == '1'

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
	<Notifications />
</template>
