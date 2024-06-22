<script>
import AjaxLoader from './AjaxLoader.vue'
import CommonMixins from '../mixins/CommonMixins'
import { mailbox } from '../stores/mailbox'

export default {
    mixins: [CommonMixins],

    components: {
        AjaxLoader,
    },

    emits: ['loadMessages'],

    data() {
        return {
            mailbox,
        }
    },

    methods: {
        loadMessages() {
            this.$emit('loadMessages')
        },

        // mark selected messages as read
        markSelectedRead() {
            if (!mailbox.selected.length) {
                return false
            }
            this.put(this.resolve(`/api/v1/messages`), { 'Read': true, 'IDs': mailbox.selected }, (response) => {
                window.scrollInPlace = true
                this.loadMessages()
            })
        },

        isSelected(id) {
            return mailbox.selected.indexOf(id) != -1
        },

        // mark selected messages as unread
        markSelectedUnread() {
            if (!mailbox.selected.length) {
                return false
            }
            this.put(this.resolve(`/api/v1/messages`), { 'Read': false, 'IDs': mailbox.selected }, (response) => {
                window.scrollInPlace = true
                this.loadMessages()
            })
        },

        // universal handler to delete current or selected messages
        deleteMessages() {
            let ids = []
            ids = JSON.parse(JSON.stringify(mailbox.selected))
            if (!ids.length) {
                return false
            }

            this.delete(this.resolve(`/api/v1/messages`), { 'IDs': ids }, (response) => {
                window.scrollInPlace = true
                this.loadMessages()
            })
        },

        // test if any selected emails are unread
        selectedHasUnread() {
            if (!mailbox.selected.length) {
                return false
            }
            for (let i in mailbox.messages) {
                if (this.isSelected(mailbox.messages[i].ID) && !mailbox.messages[i].Read) {
                    return true
                }
            }
            return false
        },

        // test of any selected emails are read
        selectedHasRead() {
            if (!mailbox.selected.length) {
                return false
            }
            for (let i in mailbox.messages) {
                if (this.isSelected(mailbox.messages[i].ID) && mailbox.messages[i].Read) {
                    return true
                }
            }
            return false
        },
    }
}
</script>

<template>
    <template v-if="mailbox.selected.length">
        <button class="list-group-item list-group-item-action" :disabled="!selectedHasUnread()"
            v-on:click="markSelectedRead">
            <i class="bi bi-eye-fill me-1"></i>
            Mark read
        </button>
        <button class="list-group-item list-group-item-action" :disabled="!selectedHasRead()"
            v-on:click="markSelectedUnread">
            <i class="bi bi-eye-slash me-1"></i>
            Mark unread
        </button>
        <button class="list-group-item list-group-item-action" v-on:click="deleteMessages()">
            <i class="bi bi-trash-fill me-1 text-danger"></i>
            Delete selected
        </button>
        <button class="list-group-item list-group-item-action" v-on:click="mailbox.selected = []">
            <i class="bi bi-x-circle me-1"></i>
            Cancel selection
        </button>
    </template>

    <AjaxLoader :loading="loading" />
</template>
