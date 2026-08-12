<script lang="ts">
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import { Icon } from '@steeze-ui/svelte-icon';
	import {
		ChartBar,
		CircleStack,
		CpuChip,
		ExclamationTriangle,
		Signal
	} from '@steeze-ui/heroicons';
	import { untrack } from 'svelte';

	type ObservabilityResponse = components['schemas']['ProjectObservabilityResponse'];
	type ServiceObservability = components['schemas']['ServiceObservability'];
	type Status = ServiceObservability['status'];
	type HistoryPoint = {
		at: string;
		cpu: number;
		memory: number;
		networkIn: number;
		networkOut: number;
	};

	const DEFAULT_REFRESH_SECONDS = 15;
	const MAX_HISTORY_POINTS = 300;
	const timeFormatter = new Intl.DateTimeFormat('en-GB', {
		hour: '2-digit',
		minute: '2-digit',
		second: '2-digit'
	});

	let projectId = $derived(page.params.id as string);
	let snapshot = $state<ObservabilityResponse | null>(null);
	let history = $state<HistoryPoint[]>([]);
	let serviceRates = $state<Record<string, number>>({});
	let loading = $state(true);
	let refreshing = $state(false);
	let loadError = $state('');
	let refreshError = $state('');
	let retryVersion = $state(0);
	let timer: ReturnType<typeof setTimeout> | null = null;

	let memoryPercent = $derived(
		snapshot?.summary.memory_limit_bytes
			? (snapshot.summary.memory_used_bytes / snapshot.summary.memory_limit_bytes) * 100
			: 0
	);
	let networkRate = $derived({
		in: history.at(-1)?.networkIn ?? 0,
		out: history.at(-1)?.networkOut ?? 0
	});
	let chartNetworkMax = $derived(
		Math.max(1, ...history.flatMap((point) => [point.networkIn, point.networkOut]))
	);
	let hasMetricHistory = $derived(history.some((point) => point.cpu > 0 || point.memory > 0));
	let hasNetworkSamples = $derived(history.length > 1);
	let hasNetworkActivity = $derived(
		history.some((point) => point.networkIn > 0 || point.networkOut > 0)
	);

	$effect(() => {
		const id = projectId;
		retryVersion;
		let cancelled = false;

		function clearRefresh() {
			if (timer) clearTimeout(timer);
			timer = null;
		}

		function scheduleRefresh() {
			clearRefresh();
			if (cancelled || document.hidden) return;
			timer = setTimeout(async () => {
				await fetchSnapshot();
				scheduleRefresh();
			}, (snapshot?.refresh_after_seconds ?? DEFAULT_REFRESH_SECONDS) * 1000);
		}

		async function fetchHistory() {
			try {
				const { data } = await api.GET('/api/projects/{id}/observability/history', {
					params: { path: { id }, query: { since: '7d', max_points: MAX_HISTORY_POINTS } }
				});
				if (cancelled || !data) return;
				history = mergeHistory(history, retainedTimeline(data));
			} catch {
				// Live metrics remain useful when retained history is unavailable.
			}
		}

		async function fetchSnapshot() {
			if (cancelled) return;
			if (snapshot) refreshing = true;
			else loading = true;
			refreshError = '';
			try {
				const previous = snapshot;
				const { data, error } = await api.GET('/api/projects/{id}/observability', {
					params: { path: { id } }
				});
				if (cancelled) return;
				if (error || !data) {
					const message =
						(error as { error?: string } | undefined)?.error ?? 'Failed to load metrics';
					if (previous) refreshError = message;
					else loadError = message;
					return;
				}

				const previousAt = previous ? new Date(previous.sampled_at).getTime() : 0;
				const currentAt = new Date(data.sampled_at).getTime();
				const seconds = previousAt ? Math.max(1, (currentAt - previousAt) / 1000) : 0;
				const previousServices = new Map(
					(previous?.services ?? []).map((service) => [service.service_id, service])
				);
				const nextRates: Record<string, number> = {};
				for (const service of data.services) {
					const prior = previousServices.get(service.service_id)?.container;
					if (!service.container || !prior || !seconds) continue;
					const inDelta = service.container.network_in_bytes_total - prior.network_in_bytes_total;
					const outDelta =
						service.container.network_out_bytes_total - prior.network_out_bytes_total;
					nextRates[service.service_id] = Math.max(0, (inDelta + outDelta) / seconds);
				}
				const inDelta = previous
					? data.summary.network_in_bytes_total - previous.summary.network_in_bytes_total
					: 0;
				const outDelta = previous
					? data.summary.network_out_bytes_total - previous.summary.network_out_bytes_total
					: 0;
				snapshot = data;
				serviceRates = nextRates;
				history = mergeHistory(history, [
					{
						at: data.sampled_at,
						cpu: data.summary.cpu_percent,
						memory: data.summary.memory_limit_bytes
							? (data.summary.memory_used_bytes / data.summary.memory_limit_bytes) * 100
							: 0,
						networkIn: seconds ? Math.max(0, inDelta / seconds) : 0,
						networkOut: seconds ? Math.max(0, outDelta / seconds) : 0
					}
				]);
			} catch {
				if (cancelled) return;
				if (snapshot) refreshError = 'Network error while refreshing metrics';
				else loadError = 'Network error';
			} finally {
				if (!cancelled) {
					loading = false;
					refreshing = false;
				}
			}
		}

		function handleVisibility() {
			clearRefresh();
			if (!document.hidden) {
				untrack(() => void fetchSnapshot().finally(scheduleRefresh));
			}
		}

		document.addEventListener('visibilitychange', handleVisibility);
		untrack(() => void fetchHistory());
		untrack(() => void fetchSnapshot().finally(scheduleRefresh));
		return () => {
			cancelled = true;
			clearRefresh();
			document.removeEventListener('visibilitychange', handleVisibility);
		};
	});

	function retry() {
		loadError = '';
		refreshError = '';
		retryVersion++;
	}

	function mergeHistory(current: HistoryPoint[], incoming: HistoryPoint[]): HistoryPoint[] {
		const points = new Map(current.map((point) => [point.at, point]));
		for (const point of incoming) points.set(point.at, point);
		return [...points.values()]
			.sort((left, right) => new Date(left.at).getTime() - new Date(right.at).getTime())
			.slice(-MAX_HISTORY_POINTS);
	}

	function retainedTimeline(
		response: components['schemas']['ProjectObservabilityHistoryResponse']
	): HistoryPoint[] {
		type Aggregate = {
			at: string;
			cpu: number;
			memoryUsed: number;
			memoryLimit: number;
			networkIn: number;
			networkOut: number;
		};
		const byTime = new Map<string, Aggregate>();
		for (const service of response.services) {
			for (const deployment of service.deployments) {
				const points = [...deployment.points].sort(
					(left, right) => new Date(left.sampled_at).getTime() - new Date(right.sampled_at).getTime()
				);
				for (let index = 0; index < points.length; index++) {
					const point = points[index];
					const previous = points[index - 1];
					const seconds = previous
						? Math.max(1, (new Date(point.sampled_at).getTime() - new Date(previous.sampled_at).getTime()) / 1000)
						: 0;
					const bucketAt = new Date(
						Math.floor(new Date(point.sampled_at).getTime() / 60_000) * 60_000
					).toISOString();
					const aggregate = byTime.get(bucketAt) ?? {
						at: bucketAt,
						cpu: 0,
						memoryUsed: 0,
						memoryLimit: 0,
						networkIn: 0,
						networkOut: 0
					};
					aggregate.cpu += point.cpu_percent;
					aggregate.memoryUsed += point.memory_used_bytes;
					aggregate.memoryLimit += point.memory_limit_bytes;
					if (previous && seconds) {
						aggregate.networkIn += Math.max(
							0,
							(point.network_in_bytes_total - previous.network_in_bytes_total) / seconds
						);
						aggregate.networkOut += Math.max(
							0,
							(point.network_out_bytes_total - previous.network_out_bytes_total) / seconds
						);
					}
					byTime.set(bucketAt, aggregate);
				}
			}
		}
		return [...byTime.values()].map((point) => ({
			at: point.at,
			cpu: point.cpu,
			memory: point.memoryLimit ? (point.memoryUsed / point.memoryLimit) * 100 : 0,
			networkIn: point.networkIn,
			networkOut: point.networkOut
		}));
	}

	function formatBytes(bytes: number, perSecond = false): string {
		const suffix = perSecond ? '/s' : '';
		if (bytes < 1024) return `${Math.round(bytes)} B${suffix}`;
		if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} KB${suffix}`;
		if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} MB${suffix}`;
		return `${(bytes / 1024 ** 3).toFixed(1)} GB${suffix}`;
	}

	function formatPercent(value: number): string {
		return `${value.toFixed(value >= 10 ? 0 : 1)}%`;
	}

	function formatTime(value: string | undefined): string {
		return value ? timeFormatter.format(new Date(value)) : '—';
	}

	function formatUptime(seconds: number): string {
		if (seconds < 60) return '< 1m';
		const days = Math.floor(seconds / 86_400);
		const hours = Math.floor((seconds % 86_400) / 3_600);
		const minutes = Math.floor((seconds % 3_600) / 60);
		if (days) return `${days}d ${hours}h`;
		if (hours) return `${hours}h ${minutes}m`;
		return `${minutes}m`;
	}

	function statusLabel(status: Status): string {
		return {
			not_deployed: 'Not deployed',
			running: 'Running',
			stopped: 'Stopped',
			unreachable: 'Unreachable',
			error: 'Failed'
		}[status];
	}

	function statusTone(status: Status): 'neutral' | 'success' | 'warning' | 'danger' {
		if (status === 'running') return 'success';
		if (status === 'not_deployed') return 'neutral';
		if (status === 'stopped') return 'warning';
		return 'danger';
	}

	function healthTone(
		degradedServices: number,
		runningServices: number
	): 'neutral' | 'success' | 'warning' {
		if (degradedServices > 0) return 'warning';
		if (runningServices > 0) return 'success';
		return 'neutral';
	}

	function healthSummary(degradedServices: number, runningServices: number): string {
		if (degradedServices > 0) {
			return `${degradedServices} ${degradedServices === 1 ? 'service needs' : 'services need'} attention`;
		}
		if (runningServices > 0) return 'Active containers are reporting normally';
		return 'No active containers are reporting metrics';
	}

	function chartPath(values: number[], maxValue: number): string {
		if (!values.length) return '';
		const width = 720;
		const height = 180;
		const step = values.length === 1 ? width : width / (values.length - 1);
		return values
			.map((value, index) => {
				const x = index * step;
				const y = height - Math.min(1, Math.max(0, value / maxValue)) * height;
				return `${index ? 'L' : 'M'}${x.toFixed(1)} ${y.toFixed(1)}`;
			})
			.join(' ');
	}

	function chartPoints(values: number[], maxValue: number): string {
		if (!values.length) return '';
		const width = 720;
		const height = 180;
		const step = values.length === 1 ? width : width / (values.length - 1);
		return values
			.map((value, index) => {
				const x = index * step;
				const y = height - Math.min(1, Math.max(0, value / maxValue)) * height;
				return `${x.toFixed(1)},${y.toFixed(1)}`;
			})
			.join(' ');
	}

	function serviceNetworkRate(service: ServiceObservability): number {
		return serviceRates[service.service_id] ?? 0;
	}
</script>

<svelte:head>
	<title>Project observability · Uploy</title>
</svelte:head>

<div class="min-h-0 flex-1 overflow-y-auto rounded-xl border border-border bg-card">
	<div class="mx-auto w-full max-w-6xl px-5 py-8 sm:px-8 sm:py-10">
		<header class="mb-7 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
			<div class="max-w-2xl">
				<h1 class="text-2xl font-semibold tracking-[-0.02em] text-foreground">
					Project observability
				</h1>
				<p class="mt-2 text-sm leading-relaxed text-muted-foreground">
					Live health and resource usage for active deployment containers across every environment.
				</p>
			</div>
			<div class="flex flex-wrap items-center gap-2">
				<Badge tone={snapshot ? 'success' : 'neutral'}>
					<span class="mr-1.5 inline-block h-1.5 w-1.5 rounded-full bg-current"></span>
					{snapshot ? 'Live' : 'Connecting'}
				</Badge>
				<span class="rounded-md border border-border px-2.5 py-1 text-xs text-muted-foreground">
					{refreshing
						? 'Refreshing…'
						: `Updates every ${snapshot?.refresh_after_seconds ?? DEFAULT_REFRESH_SECONDS}s`}
				</span>
			</div>
		</header>

		{#if loading && !snapshot}
			<div class="space-y-5" aria-label="Loading observability" aria-busy="true">
				<div
					class="grid gap-px overflow-hidden rounded-xl border border-border bg-border sm:grid-cols-3"
				>
					{#each Array(3)}
						<div class="h-28 animate-pulse bg-card"></div>
					{/each}
				</div>
				<div class="h-72 animate-pulse rounded-xl border border-border bg-muted/30"></div>
			</div>
		{:else if loadError && !snapshot}
			<EmptyState icon={ExclamationTriangle} title="Metrics unavailable" description={loadError}>
				{#snippet actions()}
					<Button size="sm" variant="secondary" onclick={retry}>Try again</Button>
				{/snippet}
			</EmptyState>
		{:else if snapshot && snapshot.summary.total_services === 0}
			<EmptyState
				icon={ChartBar}
				title="No services in this project"
				description="Add a service to see container health and resource usage here."
			/>
		{:else if snapshot}
			<section
				class="overflow-hidden rounded-xl border border-border motion-safe:animate-slide-up-fade"
				aria-label="Project health"
			>
				<header
					class="flex flex-col gap-3 border-b border-border bg-muted/30 px-5 py-4 sm:flex-row sm:items-center sm:justify-between"
				>
					<div class="flex min-w-0 items-start gap-3">
						<span
							class="grid h-8 w-8 flex-none place-items-center rounded-md bg-card text-primary-deep ring-1 ring-border"
						>
							<Icon src={ChartBar} theme="outline" class="h-4 w-4" />
						</span>
						<div class="min-w-0">
							<h2 class="text-[15px] font-semibold text-foreground">Project health</h2>
							<p class="mt-0.5 text-xs leading-relaxed text-muted-foreground">
								{snapshot.summary.running_services} of {snapshot.summary.total_services} services are
								contributing live metrics.
							</p>
						</div>
					</div>
					<Badge
						tone={healthTone(snapshot.summary.degraded_services, snapshot.summary.running_services)}
					>
						{healthSummary(snapshot.summary.degraded_services, snapshot.summary.running_services)}
					</Badge>
				</header>
				<div class="grid sm:grid-cols-3">
					<div class="border-b border-border px-5 py-5 sm:border-r sm:border-b-0">
						<div class="flex items-center justify-between gap-3">
							<p class="text-xs font-medium text-muted-foreground">CPU usage</p>
							<Icon src={CpuChip} theme="outline" class="h-4 w-4 text-info" />
						</div>
						<p
							class="mt-3 font-mono text-[1.75rem] leading-none font-semibold tracking-[-0.03em] text-foreground tabular-nums"
						>
							{formatPercent(snapshot.summary.cpu_percent)}
						</p>
						<p class="mt-2 text-xs text-muted-foreground">Sum across running containers</p>
					</div>
					<div class="border-b border-border px-5 py-5 sm:border-r sm:border-b-0">
						<div class="flex items-center justify-between gap-3">
							<p class="text-xs font-medium text-muted-foreground">Memory</p>
							<Icon src={CircleStack} theme="outline" class="h-4 w-4 text-success" />
						</div>
						<p
							class="mt-3 font-mono text-[1.75rem] leading-none font-semibold tracking-[-0.03em] text-foreground tabular-nums"
						>
							{formatBytes(snapshot.summary.memory_used_bytes)}
						</p>
						<p class="mt-2 text-xs text-muted-foreground">
							{formatPercent(memoryPercent)} of {formatBytes(snapshot.summary.memory_limit_bytes)} allocated
						</p>
					</div>
					<div class="px-5 py-5">
						<div class="flex items-center justify-between gap-3">
							<p class="text-xs font-medium text-muted-foreground">Network rate</p>
							<Icon src={Signal} theme="outline" class="h-4 w-4 text-primary-deep" />
						</div>
						<p
							class="mt-3 font-mono text-[1.75rem] leading-none font-semibold tracking-[-0.03em] text-foreground tabular-nums"
						>
							{formatBytes(networkRate.in + networkRate.out, true)}
						</p>
						<p class="mt-2 text-xs text-muted-foreground">
							{formatBytes(networkRate.in, true)} received · {formatBytes(networkRate.out, true)} sent
						</p>
					</div>
				</div>
				<footer
					class="flex flex-wrap items-center justify-between gap-3 border-t border-border px-5 py-3 text-xs text-muted-foreground"
				>
					<span>Active deployments only · all environments</span>
					<time datetime={snapshot.sampled_at} class="font-mono tabular-nums">
						Sampled {formatTime(snapshot.sampled_at)}
					</time>
				</footer>
			</section>

			<section
				class="mt-5 overflow-hidden rounded-xl border border-border"
				aria-labelledby="history-title"
			>
				<header
					class="flex flex-col gap-3 px-5 py-4 sm:flex-row sm:items-center sm:justify-between"
				>
					<div>
						<h2 id="history-title" class="text-[15px] font-semibold text-foreground">
							Retained history
						</h2>
						<p class="mt-0.5 text-xs text-muted-foreground">
							Up to seven days from servers with monitoring enabled
						</p>
					</div>
					<div class="flex flex-wrap gap-3 text-xs text-muted-foreground">
						<span class="flex items-center gap-1.5"
							><span class="h-1.5 w-1.5 rounded-full bg-info"></span>CPU</span
						>
						<span class="flex items-center gap-1.5"
							><span class="h-1.5 w-1.5 rounded-full bg-success"></span>Memory</span
						>
						<span class="flex items-center gap-1.5"
							><span class="h-1.5 w-1.5 rounded-full bg-primary-deep"></span>Network</span
						>
					</div>
				</header>
				<div class="border-t border-border px-5 py-5">
					{#if hasMetricHistory}
						<svg
							viewBox="0 0 720 180"
							class="h-52 w-full text-border"
							role="img"
							aria-label="Retained CPU and memory usage"
							preserveAspectRatio="none"
						>
							<path
								d="M0 0V180M180 0V180M360 0V180M540 0V180M720 0V180M0 0H720M0 90H720M0 180H720"
								stroke="currentColor"
								stroke-width="1"
							/>
							<path
								d={chartPath(
									history.map((point) => point.cpu),
									100
								)}
								class="text-info"
								fill="none"
								stroke="currentColor"
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="3"
							/>
							<path
								d={chartPath(
									history.map((point) => point.memory),
									100
								)}
								class="text-success"
								fill="none"
								stroke="currentColor"
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="3"
							/>
						</svg>
						<div
							class="mt-3 flex items-center justify-between gap-3 text-[11px] text-muted-foreground tabular-nums"
						>
							<span>{formatTime(history[0]?.at)}</span>
							<span
								>CPU {formatPercent(snapshot.summary.cpu_percent)} · Memory {formatPercent(
									memoryPercent
								)}</span
							>
							<span>{formatTime(history.at(-1)?.at)}</span>
						</div>
					{:else}
						<div
							class="grid min-h-52 place-items-center border border-dashed border-border bg-muted/20 px-6 text-center"
						>
							<div class="max-w-sm">
								<Icon
									src={ChartBar}
									theme="outline"
									class="mx-auto h-5 w-5 text-muted-foreground"
								/>
								<p class="mt-3 text-sm font-medium text-foreground">
									Waiting for container metrics
								</p>
								<p class="mt-1 text-xs leading-relaxed text-muted-foreground">
									{snapshot.summary.running_services
									? 'Enable server monitoring to retain this timeline across page visits.'
									: 'Start or restore an active deployment to populate this chart.'}
								</p>
							</div>
						</div>
					{/if}
				</div>
				<div class="border-t border-border px-5 py-5">
					<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
						<div>
							<h3 class="text-sm font-semibold text-foreground">Network throughput</h3>
							<p class="mt-0.5 text-xs text-muted-foreground">
								Rate calculated between successive retained Docker counters.
							</p>
						</div>
						<div class="flex items-center gap-4 text-xs text-muted-foreground tabular-nums">
							<span class="flex items-center gap-1.5"
								><span class="h-1.5 w-1.5 rounded-full bg-primary-deep"></span>{formatBytes(
									networkRate.in,
									true
								)} in</span
							>
							<span class="flex items-center gap-1.5"
								><span class="h-1.5 w-1.5 rounded-full bg-warning"></span>{formatBytes(
									networkRate.out,
									true
								)} out</span
							>
						</div>
					</div>
					{#if hasNetworkActivity}
						<svg
							viewBox="0 0 720 180"
							class="mt-5 h-40 w-full text-border"
							role="img"
							aria-label="Retained network throughput"
							preserveAspectRatio="none"
						>
							<path
								d="M0 0V180M180 0V180M360 0V180M540 0V180M720 0V180M0 0H720M0 90H720M0 180H720"
								stroke="currentColor"
								stroke-width="1"
							/>
							<polyline
								points={chartPoints(
									history.map((point) => point.networkIn),
									chartNetworkMax
								)}
								class="text-primary-deep"
								fill="none"
								stroke="currentColor"
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="3"
							/>
							<polyline
								points={chartPoints(
									history.map((point) => point.networkOut),
									chartNetworkMax
								)}
								class="text-warning"
								fill="none"
								stroke="currentColor"
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="3"
							/>
						</svg>
					{:else}
						<div
							class="mt-5 flex min-h-40 items-center justify-center border border-dashed border-border bg-muted/20 px-6 text-center text-xs leading-relaxed text-muted-foreground"
						>
							{hasNetworkSamples
								? 'No network traffic has been observed in retained samples.'
								: `Waiting for the next ${snapshot?.refresh_after_seconds ?? DEFAULT_REFRESH_SECONDS}-second sample to calculate throughput.`}
						</div>
					{/if}
				</div>
			</section>

			<section
				class="mt-5 overflow-hidden rounded-xl border border-border"
				aria-labelledby="services-title"
			>
				<header
					class="flex flex-col gap-3 px-5 py-4 sm:flex-row sm:items-center sm:justify-between"
				>
					<div>
						<h2 id="services-title" class="text-[15px] font-semibold text-foreground">
							Services and containers
						</h2>
						<p class="mt-0.5 text-xs text-muted-foreground">
							Every service remains visible, even when its active deployment cannot report metrics.
						</p>
					</div>
					<Badge
						tone={healthTone(snapshot.summary.degraded_services, snapshot.summary.running_services)}
						variant="outline"
					>
						{snapshot.summary.running_services} running · {snapshot.summary.degraded_services}
						degraded
					</Badge>
				</header>
				<div class="sm:hidden">
					{#each snapshot.services as service (service.service_id)}
						<article class="border-t border-border px-5 py-4">
							<div class="flex items-start justify-between gap-3">
								<div class="min-w-0">
									<a
										href="/services/{service.service_id}"
										class="block truncate text-sm font-medium text-foreground underline-offset-4 hover:underline focus-visible:rounded-sm focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
										>{service.name}</a
									>
									<p class="mt-1 truncate font-mono text-[11px] text-muted-foreground">
										{service.container?.name ?? 'No active container'} · {service.environment_name ||
											'Unknown environment'}
									</p>
								</div>
								<Badge tone={statusTone(service.status)}>{statusLabel(service.status)}</Badge>
							</div>
							{#if service.error}
								<p class="mt-3 text-xs leading-relaxed text-destructive">{service.error}</p>
							{/if}
							<dl class="mt-4 grid grid-cols-3 gap-3 border-t border-border pt-3 text-xs">
								<div>
									<dt class="text-muted-foreground">CPU</dt>
									<dd class="mt-1 font-mono text-[13px] font-medium text-foreground tabular-nums">
										{service.container ? formatPercent(service.container.cpu_percent) : '—'}
									</dd>
								</div>
								<div>
									<dt class="text-muted-foreground">Memory</dt>
									<dd class="mt-1 font-mono text-[13px] font-medium text-foreground tabular-nums">
										{service.container ? formatBytes(service.container.memory_used_bytes) : '—'}
									</dd>
								</div>
								<div>
									<dt class="text-muted-foreground">Network</dt>
									<dd class="mt-1 font-mono text-[13px] font-medium text-foreground tabular-nums">
										{service.container ? formatBytes(serviceNetworkRate(service), true) : '—'}
									</dd>
								</div>
							</dl>
						</article>
					{/each}
				</div>
				<div class="hidden overflow-x-auto border-t border-border sm:block">
					<table class="w-full min-w-[760px] text-left text-sm">
						<thead class="bg-muted/40 text-xs text-muted-foreground">
							<tr>
								<th class="px-5 py-2.5 font-medium">Service</th>
								<th class="px-4 py-2.5 font-medium">Environment</th>
								<th class="px-4 py-2.5 font-medium">Status</th>
								<th class="px-4 py-2.5 text-right font-medium">CPU</th>
								<th class="px-4 py-2.5 text-right font-medium">Memory</th>
								<th class="px-5 py-2.5 text-right font-medium">Network</th>
							</tr>
						</thead>
						<tbody>
							{#each snapshot.services as service (service.service_id)}
								<tr
									class="border-t border-border transition-colors first:border-t-0 hover:bg-muted/25"
								>
									<td class="max-w-[300px] px-5 py-3.5">
										<a
											href="/services/{service.service_id}"
											class="block truncate font-medium text-foreground underline-offset-4 hover:underline focus-visible:rounded-sm focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
											>{service.name}</a
										>
										<p class="mt-1 truncate font-mono text-[11px] text-muted-foreground">
											{service.container
												? `${service.container.name} · up ${formatUptime(service.container.uptime_seconds)}`
												: 'No active container'}
										</p>
										{#if service.error}<p
												class="mt-1 truncate text-xs text-destructive"
												title={service.error}
											>
												{service.error}
											</p>{/if}
									</td>
									<td class="px-4 py-3.5 text-xs text-muted-foreground"
										>{service.environment_name || 'Unknown'}</td
									>
									<td class="px-4 py-3.5"
										><Badge tone={statusTone(service.status)}>{statusLabel(service.status)}</Badge
										></td
									>
									<td
										class="px-4 py-3.5 text-right font-mono text-[13px] text-foreground tabular-nums"
										>{service.container ? formatPercent(service.container.cpu_percent) : '—'}</td
									>
									<td
										class="px-4 py-3.5 text-right font-mono text-[13px] text-foreground tabular-nums"
										>{service.container
											? formatBytes(service.container.memory_used_bytes)
											: '—'}</td
									>
									<td
										class="px-5 py-3.5 text-right font-mono text-[13px] text-foreground tabular-nums"
										>{service.container ? formatBytes(serviceNetworkRate(service), true) : '—'}</td
									>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</section>
		{/if}

		{#if refreshError && snapshot}
			<div
				class="mt-4 flex flex-wrap items-center justify-between gap-3 rounded-lg border border-warning/30 bg-warning-muted px-4 py-3 text-sm text-warning"
				role="status"
			>
				<span>{refreshError} Showing the last successful sample.</span>
				<Button size="sm" variant="secondary" onclick={retry}>Retry</Button>
			</div>
		{/if}
	</div>
</div>
