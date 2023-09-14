<script>
import { RouterLink, RouterView } from 'vue-router'
import CommonMixins from './mixins/CommonMixins.js'
// import WebsocketMixin from './mixins/WebsocketMixin.js'
import Notifications from "./components/Notifications.vue"
import { mailbox } from "./stores/mailbox"
// import RepoSelector from "./components/RepoSelector.vue"
// import ThemeToggle from "./components/ThemeToggle.vue"
// import Loading from './components/Loading.vue'

export default {
	mixins: [CommonMixins],

	components: {
		Notifications,
	},

	beforeMount() {
		document.title = document.title + ' - ' + location.hostname
		window.baseURL = this.$router.resolve(`/`).href
		mailbox.showTagColors = localStorage.getItem('showTagsColors') == '1'
	},

	mounted() {
	},

}
</script>

<template>
	<RouterView />
	<Notifications />
</template>
