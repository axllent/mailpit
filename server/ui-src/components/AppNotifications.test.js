import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { runInNewContext } from "node:vm";
import { parse } from "@vue/compiler-sfc";
import { transform } from "esbuild";
import { createMemoryHistory, createRouter } from "vue-router";

const { descriptor } = parse(readFileSync(new URL("./AppNotifications.vue", import.meta.url), "utf8"));
const { code } = await transform(descriptor.script.content, { format: "cjs" });

const setup = (permission = "granted", base = "/") => {
	const notifications = [];
	let focused = false;
	let socket;
	const mailbox = { tags: [], notificationsEnabled: permission === "granted" };
	const router = createRouter({
		history: createMemoryHistory(base),
		routes: [
			{ path: "/view/:id", component: {} },
			{ path: "/:pathMatch(.*)*", component: {} },
		],
	});
	const imports = {
		"../mixins/CommonMixins": { default: {} },
		bootstrap: { Toast: {} },
		"../stores/mailbox": { mailbox },
		"../stores/pagination": { pagination: {} },
	};
	const context = {
		module: { exports: {} },
		require: (id) => imports[id],
		window: {
			Notification: true,
			focus: () => {
				focused = true;
			},
			setTimeout: () => {},
		},
		Notification: class {
			static permission = permission;

			constructor(title, options) {
				this.title = title;
				this.options = options;
				this.closed = false;
				notifications.push(this);
			}

			close() {
				this.closed = true;
			}
		},
		WebSocket: class {
			constructor() {
				socket = this;
			}
		},
	};
	runInNewContext(code, context);
	const component = context.module.exports.default;
	const instance = {
		...component.data(),
		...component.methods,
		$router: router,
		resolve: (path) => router.resolve(path).href,
		eventBus: { emit: () => {} },
		setMessageToast: () => {},
	};
	return { instance, notifications, context, router, isFocused: () => focused, getSocket: () => socket };
};

for (const base of ["/", "/mailpit/"]) {
	test(`notification click focuses the tab and opens its message with base ${base}`, async () => {
		const { instance, notifications, router, isFocused } = setup("granted", base);
		instance.browserNotify("New mail from: sender@example.com", "Subject", "message-1");
		const notification = notifications[0];
		assert.equal(notification.title, "New mail from: sender@example.com");
		assert.equal(notification.options.body, "Subject");
		assert.equal(notification.options.icon, `${base}notification.png`);
		assert.equal(isFocused(), false);

		let prevented = false;
		notification.onclick({
			preventDefault: () => {
				prevented = true;
			},
		});
		await router.isReady();
		// the default must not be prevented, or Firefox will not focus the tab / raise the window
		assert.equal(prevented, false);
		assert.equal(isFocused(), true);
		assert.equal(notification.closed, true);
		assert.equal(router.currentRoute.value.path, "/view/message-1");
		assert.equal(router.resolve(router.currentRoute.value).href, `${base}view/message-1`);
	});
}

test("each notification opens its own message", async () => {
	const { instance, notifications, router } = setup();
	instance.browserNotify("First message", "First subject", "first");
	instance.browserNotify("Second message", "Second subject", "second");
	notifications[0].onclick({ preventDefault: () => {} });
	await router.isReady();
	assert.equal(router.currentRoute.value.path, "/view/first");
	notifications[1].onclick({ preventDefault: () => {} });
	await new Promise((resolve) => router.afterEach(resolve));
	assert.equal(router.currentRoute.value.path, "/view/second");
});

test("new-message events pass the message ID to the notification", async () => {
	const { instance, notifications, router, getSocket } = setup();
	instance.connect();
	getSocket().onmessage({
		data: JSON.stringify({
			Type: "new",
			Data: { ID: "received", From: { Address: "sender@example.com" }, Subject: "s".repeat(120), Tags: [] },
		}),
	});
	assert.equal(notifications.length, 1);
	assert.equal(notifications[0].options.body.length, 100);
	notifications[0].onclick({ preventDefault: () => {} });
	await router.isReady();
	assert.equal(router.currentRoute.value.path, "/view/received");
});

for (const permission of ["default", "denied"]) {
	test(`does not notify when permission is ${permission}`, () => {
		const { instance, notifications } = setup(permission);
		instance.browserNotify("Title", "Body", "message");
		assert.equal(notifications.length, 0);
	});
}

test("does not notify when browser notifications are unsupported", () => {
	const { instance, notifications, context } = setup();
	delete context.window.Notification;
	delete context.Notification;
	instance.browserNotify("Title", "Body", "message");
	assert.equal(notifications.length, 0);
});
