<script>
import { mailbox } from "../stores/mailbox.js";

export default {
	data() {
		return {
			updating: false,
			needsUpdate: false,
			timeout: 500,
		};
	},

	computed: {
		mailboxUnread() {
			return mailbox.unread;
		},
	},

	watch: {
		mailboxUnread: {
			handler() {
				if (this.updating) {
					this.needsUpdate = true;
					return;
				}

				this.scheduleUpdate();
			},
			immediate: true,
		},
	},

	methods: {
		scheduleUpdate() {
			this.updating = true;
			this.needsUpdate = false;

			window.setTimeout(() => {
				this.updateAppBadge();
				this.updating = false;

				if (this.needsUpdate) {
					this.scheduleUpdate();
				}
			}, this.timeout);
		},

		updateAppBadge() {
			if (!("setAppBadge" in navigator)) {
				return;
			}

			navigator.setAppBadge(this.mailboxUnread);
		},
	},

	render() {
		// to remove webkit warnings about missing template or render function
		return false;
	},
};
</script>
