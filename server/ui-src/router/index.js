import { createRouter, createWebHistory } from 'vue-router'
import MailboxView from '../views/MailboxView.vue'
import SearchView from '../views/SearchView.vue'
import NotFoundView from '../views/NotFoundView.vue'
// import EditView from '../views/EditView.vue'
// import StatsView from '../views/StatsView.vue'
// import NotFound from '../views/NotFound.vue'

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
			// name: 'home',
			component: MailboxView
		},
		{
			path: '/search',
			// name: 'edit',
			component: SearchView
		},
		// {
		//     path: '/view/:id',
		//     name: 'view',
		//     component: StatsView
		// },
		{
			path: '/:pathMatch(.*)*',
			name: 'NotFound',
			component: NotFoundView
		}
	]
})

export default router
