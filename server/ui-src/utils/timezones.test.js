import assert from "node:assert/strict";
import test from "node:test";
import { gmtOffset, timezoneOptions } from "./timezones.js";

const january = new Date("2026-01-15T12:00:00Z");
const july = new Date("2026-07-15T12:00:00Z");

test("gmtOffset: follows daylight saving for the given date", () => {
	assert.equal(gmtOffset("America/Toronto", january), "GMT-05:00");
	assert.equal(gmtOffset("America/Toronto", july), "GMT-04:00");
});

test("gmtOffset: UTC and half-hour offsets", () => {
	assert.equal(gmtOffset("UTC", july), "GMT+00:00");
	assert.equal(gmtOffset("Asia/Kolkata", july), "GMT+05:30");
});

test("gmtOffset: unknown zone returns null", () => {
	assert.equal(gmtOffset("Not/AZone", july), null);
});

test("timezoneOptions: replaces the static offset and keeps the static label as fallback", () => {
	const list = [
		{ tzCode: "America/Toronto", label: "America/Toronto (GMT-05:00)" },
		{ tzCode: "Not/AZone", label: "Not/AZone (GMT+01:00)" },
	];
	assert.deepEqual(timezoneOptions(list, july), [
		{ tzCode: "America/Toronto", label: "America/Toronto (GMT-04:00)" },
		{ tzCode: "Not/AZone", label: "Not/AZone (GMT+01:00)" },
	]);
});

test("timezoneOptions: sorted by the current offset, not the static one", () => {
	const list = [
		{ tzCode: "Asia/Kolkata", label: "Asia/Kolkata (GMT+05:30)", utc: "+05:30" },
		{ tzCode: "America/Toronto", label: "America/Toronto (GMT-05:00)", utc: "-05:00" },
		{ tzCode: "America/Panama", label: "America/Panama (GMT-05:00)", utc: "-05:00" },
		{ tzCode: "Not/AZone", label: "Not/AZone (GMT-11:00)", utc: "-11:00" },
	];
	// July: Toronto is -04:00, Panama -05:00. The unknown zone sorts by its static offset.
	assert.deepEqual(
		timezoneOptions(list, july).map((o) => o.tzCode),
		["Not/AZone", "America/Panama", "America/Toronto", "Asia/Kolkata"],
	);
	// January: Toronto and Panama tie at -05:00 and fall back to tzCode order.
	assert.deepEqual(
		timezoneOptions(list, january).map((o) => o.tzCode),
		["Not/AZone", "America/Panama", "America/Toronto", "Asia/Kolkata"],
	);
});
