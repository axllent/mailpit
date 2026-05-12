// Wraps oidc-client-ts UserManager. Reads the library off the global
// window.__mailpitOIDC__ that the conditionally-loaded oidc-entry.js
// bundle attaches — so the main app.js bundle never imports
// oidc-client-ts directly.

let mgr = null;
let configurePromise = null;

export function oidcEnabled() {
	const el = document.getElementById("app");
	return !!(el && el.dataset.oidcIssuer && el.dataset.oidcClientId);
}

function loadLib() {
	if (window.__mailpitOIDC__) return Promise.resolve(window.__mailpitOIDC__);
	return new Promise((resolve) => {
		window.addEventListener("mp-oidc-ready", () => resolve(window.__mailpitOIDC__), { once: true });
	});
}

export function configureOIDC() {
	if (configurePromise) return configurePromise;
	configurePromise = (async () => {
		if (!oidcEnabled()) return null;
		const { UserManager, WebStorageStateStore } = await loadLib();
		const el = document.getElementById("app");
		const issuer = el.dataset.oidcIssuer;
		const clientId = el.dataset.oidcClientId;
		const webroot = el.dataset.webroot || "/";
		const origin = window.location.origin;
		mgr = new UserManager({
			authority: issuer,
			client_id: clientId,
			redirect_uri: origin + webroot + "auth/callback",
			post_logout_redirect_uri: origin + webroot,
			response_type: "code",
			scope: "openid email profile offline_access",
			userStore: new WebStorageStateStore({ store: window.localStorage }),
			automaticSilentRenew: true,
			includeIdTokenInSilentRenew: true,
		});
		mgr.events.addSilentRenewError(() => {
			// Refresh-token grant failed — forget the user so the next 401
			// triggers a full redirect via the axios response interceptor.
			mgr.removeUser();
		});
		return mgr;
	})();
	return configurePromise;
}

export async function getUser() {
	await configureOIDC();
	if (!mgr) return null;
	return mgr.getUser();
}

export async function getToken() {
	const u = await getUser();
	if (!u || u.expired) return null;
	return u.id_token;
}

export async function login(returnTo) {
	await configureOIDC();
	if (!mgr) return;
	return mgr.signinRedirect({
		state: returnTo || window.location.pathname + window.location.search,
	});
}

export async function logout() {
	await configureOIDC();
	if (!mgr) return;
	return mgr.signoutRedirect();
}

export async function handleCallback() {
	await configureOIDC();
	if (!mgr) return "/";
	const u = await mgr.signinRedirectCallback();
	const el = document.getElementById("app");
	const webroot = (el && el.dataset.webroot) || "/";
	return typeof u.state === "string" ? u.state : webroot;
}
