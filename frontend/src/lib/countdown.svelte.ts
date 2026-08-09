/**
 * A live count of the seconds left until `target`, or null when there is
 * nothing to count to.
 *
 * The timer only exists while there is a target, so a panel whose work is done
 * stops waking up once a second for the rest of the session. Call it during
 * component setup, like any rune.
 *
 * @param target reads the moment to count to, as an ISO string
 */
export function countdownTo(target: () => string | null | undefined) {
	let now = $state(Date.now());

	$effect(() => {
		if (!target()) return;
		const id = setInterval(() => (now = Date.now()), 1000);
		return () => clearInterval(id);
	});

	return {
		get seconds(): number | null {
			const iso = target();
			if (!iso) return null;
			// Never negative: a pass that overruns its slot should read as "now",
			// not count upwards into a number nobody can interpret.
			return Math.max(0, Math.round((Date.parse(iso) - now) / 1000));
		}
	};
}
