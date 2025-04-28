<script>
import { mailbox } from '../stores/mailbox.js'

export default {
    computed: {
        mailboxUnread() {
            return mailbox.unread
        }
    },

    watch: {
        mailboxUnread: {
            handler(newUnread) {
                this.updateAppBadge(newUnread)
            },
            immediate: true
        }
    },

    methods: {
        updateAppBadge(unreadCount) {
            if (!('setAppBadge' in navigator)) {
                return
            }

            navigator.setAppBadge(unreadCount)
        }
    }

}
</script>

<template></template>
