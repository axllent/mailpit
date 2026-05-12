// Standalone entry that loads oidc-client-ts and exposes it on the
// global window.__mailpitOIDC__. The HTML template only includes this
// script when OIDC is configured server-side, so the library is never
// shipped to users when OIDC is disabled.
import { UserManager, WebStorageStateStore } from "oidc-client-ts";

window.__mailpitOIDC__ = { UserManager, WebStorageStateStore };
window.dispatchEvent(new Event("mp-oidc-ready"));
