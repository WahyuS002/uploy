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
