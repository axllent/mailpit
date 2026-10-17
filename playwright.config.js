import { defineConfig } from "@playwright/test";

// End-to-end tests run against a real Mailpit binary (with the UI embedded), so build it first:
//   npm run package && go build -o mailpit .
// Use MAILPIT_BIN to point at a different binary.
const bin = process.env.MAILPIT_BIN || "./mailpit";

const server = (httpPort, smtpPort, extra = "") => ({
	// no --database flag: Mailpit then uses a throwaway store, so every run starts empty
	command: `${bin} --listen 127.0.0.1:${httpPort} --smtp 127.0.0.1:${smtpPort} ${extra}`,
	url: `http://127.0.0.1:${httpPort}${extra.includes("--webroot") ? "/mailpit" : ""}/api/v1/info`,
	reuseExistingServer: false,
	timeout: 30_000,
});

export default defineConfig({
	testDir: "./e2e",
	// all tests share one Mailpit instance per project, and each starts by emptying the mailbox
	fullyParallel: false,
	workers: 1,
	forbidOnly: !!process.env.CI,
	// retries would hide flakiness, so we do not use them
	retries: 0,
	reporter: process.env.CI ? [["list"], ["github"], ["html", { open: "never" }]] : "list",
	use: { trace: "retain-on-failure" },
	// patterns are matched against the absolute path, so anchor them to the file name
	projects: [
		{ name: "default", testIgnore: /[\\/]webroot\.spec\.js$/, use: { baseURL: "http://127.0.0.1:18025/" } },
		{ name: "webroot", testMatch: /[\\/]webroot\.spec\.js$/, use: { baseURL: "http://127.0.0.1:18026/mailpit/" } },
	],
	webServer: [server(18025, 11025), server(18026, 11026, "--webroot /mailpit/")],
});
