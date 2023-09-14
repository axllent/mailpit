// State Management

import { reactive, watch } from 'vue'
import Tinycon from 'tinycon'

Tinycon.setOptions({
	height: 11,
	background: '#dd0000',
	fallback: false
})

// global mailbox info
export const mailbox = reactive({
	total: 0, 				// total number of messages
	unread: 0, 				// total unread
	count: 0, 				// total in mailbox or search
	messages: [],			// current messages
	tags: [], 				// all tags
	showTagColors: false, 	// show tag colors?
	selected: [], 			// currently selected
	connected: false, 		// websocket connection
	searching: false,		// whether we are currently searching
	refresh: false, 		// to listen from MessagesMixin
	notificationsSupported: false,
	notificationsEnabled: false,
})

watch(
	() => mailbox.total,
	(v) => {
		if (v == 0) {
			Tinycon.reset()
		} else {
			Tinycon.setBubble(v)
		}
	}
)

watch(
	() => mailbox.showTagColors,
	(v) => {
		if (v) {
			localStorage.setItem('showTagsColors', '1')
		} else {
			localStorage.removeItem('showTagsColors')
		}
	}
)
