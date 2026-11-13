// Timezone option labels for the settings select.
//
// timezones-list ships a fixed offset per zone (e.g. "America/Edmonton (GMT-07:00)"), which is wrong
// whenever daylight saving is in effect or a region changes its rules. The label is therefore built
// from the browser's own timezone data for the given date, falling back to the static label if the
// browser does not know the zone.

// Return the "GMT±hh:mm" offset of timeZone at date, or null if the zone is unknown.
export function gmtOffset(timeZone, date = new Date()) {
	try {
		const part = new Intl.DateTimeFormat("en-US", { timeZone, timeZoneName: "longOffset" })
			.formatToParts(date)
			.find((p) => p.type === "timeZoneName");
		if (!part) {
			return null;
		}
		// UTC itself is formatted as a bare "GMT"
		return part.value === "GMT" ? "GMT+00:00" : part.value;
	} catch {
		return null;
	}
}

// Map timezones-list entries to { tzCode, label } with the current offset in the label.
export function timezoneOptions(timezones, date = new Date()) {
	return timezones.map((t) => {
		const offset = gmtOffset(t.tzCode, date);
		return { tzCode: t.tzCode, label: offset ? `${t.tzCode} (${offset})` : t.label };
	});
}
