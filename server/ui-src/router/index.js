import { createRouter, createWebHistory } from "vue-router";
import AuthCallbackView from "../views/AuthCallbackView.vue";
import MailboxView from "../views/MailboxView.vue";
import MessageView from "../views/MessageView.vue";
import NotFoundView from "../views/NotFoundView.vue";
import SearchView from "../views/SearchView.vue";
import { configureOIDC, getUser, login, oidcEnabled } from "../services/oidcAuth";

const d = document.getElementById("app");
let webroot = "/";
if (d) {
	webroot = d.dataset.webroot;
}

// Resolves once oidc-client-ts is loaded (or immediately to null when OIDC is disabled).
const oidcReady = configureOIDC();

// paths are relative to webroot
const router = createRouter({
	history: createWebHistory(webroot),
	routes: [
		{
			path: "/",
			component: MailboxView,
		},
		{
			path: "/search",
			component: SearchView,
		},
		{
			path: "/view/:id",
			component: MessageView,
		},
		{
			path: "/auth/callback",
			component: AuthCallbackView,
		},
		{
			path: "/:pathMatch(.*)*",
			name: "NotFound",
			component: NotFoundView,
		},
	],
});

router.beforeEach(async (to) => {
	if (!oidcEnabled()) return true;
	await oidcReady;
	if (to.path === "/auth/callback") return true;
	const u = await getUser();
	if (u && !u.expired) return true;
	await login(to.fullPath);
	return false; // navigation paused — browser is being redirected
});

export default router;
