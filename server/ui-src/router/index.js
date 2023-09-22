import { createRouter, createWebHistory } from 'vue-router'
import MailboxView from '../views/MailboxView.vue'
import MessageView from '../views/MessageView.vue'
import NotFoundView from '../views/NotFoundView.vue'
import SearchView from '../views/SearchView.vue'

let d = document.getElementById('app')
let webroot = '/'
if (d) {
	webroot = d.dataset.webroot
}

// paths are relative to webroot
const router = createRouter({
	history: createWebHistory(webroot),
	routes: [
		{
			path: '/',
			component: MailboxView
		},
		{
			path: '/search',
			component: SearchView
		},
		{
			path: '/view/:id',
			component: MessageView
		},
		{
			path: '/:pathMatch(.*)*',
			name: 'NotFound',
			component: NotFoundView
		}
	]
})

export default router
