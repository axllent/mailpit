// Encode the message ID and asset URL in the query format used by the proxy handler.
export function screenshotProxyURL(proxy, messageID, assetURL) {
	const url = new URL(proxy);
	url.searchParams.set("data", btoa(messageID + ":" + assetURL));
	return url.href;
}
