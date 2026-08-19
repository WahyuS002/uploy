<script lang="ts">
	import type { components } from '$lib/api/v1';

	type Point = components['schemas']['ServerMetricsResponse'];
	type Metric = 'disk' | 'io' | 'load' | 'swap';

	let {
		points,
		metric,
		label,
		value
	}: { points: Point[]; metric: Metric; label: string; value: string } = $props();

	const toneClass: Record<Metric, string> = {
		disk: 'text-warning',
		io: 'text-info',
		load: 'text-success',
		swap: 'text-destructive'
	};

	function metricValue(point: Point): number {
		switch (metric) {
			case 'disk':
				return point.disk_used_percent;
			case 'io':
				return point.disk_read_bytes_total + point.disk_write_bytes_total;
			case 'load':
				return point.load_1;
			case 'swap':
				return point.swap_used_bytes;
		}
	}

	let values = $derived.by(() => {
		const raw = points.map(metricValue);
		if (metric !== 'io') return raw;
		return raw.map((value, index) => {
			if (index === 0) return 0;
			const intervalSeconds = Math.max(
				1,
				(new Date(points[index].sampled_at).getTime() -
					new Date(points[index - 1].sampled_at).getTime()) /
					1000
			);
			return Math.max(0, (value - raw[index - 1]) / intervalSeconds);
		});
	});
	let polyline = $derived.by(() => {
		if (values.length < 2) return '';
		const minimum = Math.min(...values);
		const maximum = Math.max(...values);
		const spread = maximum - minimum || 1;
		return values
			.map((item, index) => {
				const x = 4 + (index / (values.length - 1)) * 312;
				const y = 70 - ((item - minimum) / spread) * 60;
				return `${x.toFixed(1)},${y.toFixed(1)}`;
			})
			.join(' ');
	});
</script>

<div class="min-w-0 {toneClass[metric]}">
	<div class="flex items-baseline justify-between gap-3 text-xs">
		<span class="font-medium text-foreground">{label}</span>
		<span class="text-muted-foreground tabular-nums">{value}</span>
	</div>
	{#if points.length > 1}
		<svg
			class="mt-1.5 h-12 w-full overflow-visible"
			viewBox="0 0 320 76"
			role="img"
			aria-label={`${label} history trend`}
		>
			<line x1="4" y1="10" x2="316" y2="10" stroke="currentColor" stroke-opacity="0.12" />
			<line x1="4" y1="40" x2="316" y2="40" stroke="currentColor" stroke-opacity="0.12" />
			<line x1="4" y1="70" x2="316" y2="70" stroke="currentColor" stroke-opacity="0.12" />
			<polyline
				points={polyline}
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			/>
		</svg>
	{:else}
		<p class="mt-2 text-[11px] text-muted-foreground">Collecting trend data...</p>
	{/if}
</div>
