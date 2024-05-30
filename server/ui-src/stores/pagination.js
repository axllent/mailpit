import { reactive } from 'vue'

export const pagination = reactive({
	start: 0,	// pagination offset
	limit: 50, 	// per page
	total: 0,  	// total results of current view / filter
	count: 0, 	// number of messages currently displayed
})

export const limitOptions = [25, 50, 100, 200]
