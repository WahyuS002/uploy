// Locale is pinned so every user sees the same unambiguous "15 May 2026" instead of
// whatever M/D/YYYY vs D/M/YYYY the runtime happens to pick. Time zone stays the
// runtime's, so dates read correctly for the person looking at them.
const dateFormatter = new Intl.DateTimeFormat('en-GB', {
	day: 'numeric',
	month: 'short',
	year: 'numeric'
});

const dateTimeFormatter = new Intl.DateTimeFormat('en-GB', {
	day: 'numeric',
	month: 'short',
	year: 'numeric',
	hour: '2-digit',
	minute: '2-digit'
});

/** "15 May 2026" */
export function formatDate(iso: string) {
	return dateFormatter.format(new Date(iso));
}

/** "15 May 2026, 09:42" */
export function formatDateTime(iso: string) {
	return dateTimeFormatter.format(new Date(iso));
}

const relativeFormatter = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });

// Each entry is how many of this unit fit in the next one up, so the loop can
// divide its way to the largest unit the delta still fills.
const relativeUnits: [Intl.RelativeTimeFormatUnit, number][] = [
	['second', 60],
	['minute', 60],
	['hour', 24],
	['day', 7],
	['week', 4.35],
	['month', 12]
];

/**
 * "2 hours ago" — how a deployment's age is actually read. Anything a year or
 * older falls back to the absolute date, where the exact day matters more than
 * "last year".
 */
export function formatRelativeTime(iso: string) {
	let delta = (new Date(iso).getTime() - Date.now()) / 1000;
	for (const [unit, perNext] of relativeUnits) {
		if (Math.abs(delta) < perNext) return relativeFormatter.format(Math.round(delta), unit);
		delta /= perNext;
	}
	if (Math.abs(delta) < 1) return relativeFormatter.format(Math.round(delta), 'year');
	return formatDate(iso);
}

/** "51s" / "2m 8s" — how long something took, as opposed to when it happened. */
export function formatDuration(seconds: number) {
	if (seconds < 60) return `${seconds}s`;
	return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
}
