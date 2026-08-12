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

	const REFRESH_SECONDS = 10;
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
			}, REFRESH_SECONDS * 1000);
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
				history = [
					...history,
					{
						at: data.sampled_at,
						cpu: data.summary.cpu_percent,
						memory: data.summary.memory_limit_bytes
							? (data.summary.memory_used_bytes / data.summary.memory_limit_bytes) * 100
							: 0,
						networkIn: seconds ? Math.max(0, inDelta / seconds) : 0,
						networkOut: seconds ? Math.max(0, outDelta / seconds) : 0
					}
				].slice(-MAX_HISTORY_POINTS);
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
					{refreshing ? 'Refreshing…' : `Updates every ${REFRESH_SECONDS}s`}
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
			<section class="overflow-hidden rounded-xl border border-border" aria-label="Project summary">
				<div class="grid sm:grid-cols-3">
					<div class="border-b border-border px-5 py-5 sm:border-r sm:border-b-0">
						<div class="flex items-center justify-between gap-3">
							<p class="text-xs font-medium text-muted-foreground">CPU usage</p>
							<Icon src={CpuChip} theme="outline" class="h-4 w-4 text-info" />
						</div>
						<p class="mt-3 text-2xl font-semibold tracking-[-0.02em] text-foreground tabular-nums">
							{formatPercent(snapshot.summary.cpu_percent)}
						</p>
						<p class="mt-1 text-xs text-muted-foreground">Aggregate across running containers</p>
					</div>
					<div class="border-b border-border px-5 py-5 sm:border-r sm:border-b-0">
						<div class="flex items-center justify-between gap-3">
							<p class="text-xs font-medium text-muted-foreground">Memory</p>
							<Icon src={CircleStack} theme="outline" class="h-4 w-4 text-success" />
						</div>
						<p class="mt-3 text-2xl font-semibold tracking-[-0.02em] text-foreground tabular-nums">
							{formatBytes(snapshot.summary.memory_used_bytes)}
							<span class="text-sm font-normal text-muted-foreground">
								/ {formatBytes(snapshot.summary.memory_limit_bytes)}</span
							>
						</p>
						<p class="mt-1 text-xs text-muted-foreground">
							{formatPercent(memoryPercent)} of allocated limits
						</p>
					</div>
					<div class="px-5 py-5">
						<div class="flex items-center justify-between gap-3">
							<p class="text-xs font-medium text-muted-foreground">Network rate</p>
							<Icon src={Signal} theme="outline" class="h-4 w-4 text-primary-deep" />
						</div>
						<p class="mt-3 text-2xl font-semibold tracking-[-0.02em] text-foreground tabular-nums">
							{formatBytes(networkRate.in + networkRate.out, true)}
						</p>
						<p class="mt-1 text-xs text-muted-foreground">
							{formatBytes(networkRate.in, true)} in · {formatBytes(networkRate.out, true)} out
						</p>
					</div>
				</div>
				<div
					class="flex flex-wrap items-center justify-between gap-3 border-t border-border px-5 py-3 text-xs"
				>
					<span class="text-muted-foreground">
						<span class="font-medium text-foreground">{snapshot.summary.running_services}</span> of {snapshot
							.summary.total_services} services running
					</span>
					<span
						class={snapshot.summary.degraded_services ? 'text-warning' : 'text-muted-foreground'}
					>
						{snapshot.summary.degraded_services} degraded · sampled {formatTime(
							snapshot.sampled_at
						)}
					</span>
				</div>
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
							Session history
						</h2>
						<p class="mt-0.5 text-xs text-muted-foreground">
							Samples collected while this page stays open
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
					<svg
						viewBox="0 0 720 180"
						class="h-56 w-full text-border"
						role="img"
						aria-label="CPU and memory usage during this session"
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
					<div class="mt-3 flex justify-between text-[11px] text-muted-foreground tabular-nums">
						<span>{history.length ? formatTime(history[0].at) : 'Waiting'}</span><span
							>CPU {formatPercent(snapshot.summary.cpu_percent)}</span
						><span>Memory {formatPercent(memoryPercent)}</span><span>{history.length} samples</span>
					</div>
				</div>
				<div class="border-t border-border px-5 py-5">
					<div class="mb-3 flex items-center justify-between gap-3">
						<div>
							<h3 class="text-sm font-semibold text-foreground">Network throughput</h3>
							<p class="mt-0.5 text-xs text-muted-foreground">
								Calculated from cumulative Docker counters
							</p>
						</div>
						<span class="text-xs text-muted-foreground tabular-nums"
							>{formatBytes(networkRate.in + networkRate.out, true)}</span
						>
					</div>
					<svg
						viewBox="0 0 720 180"
						class="h-40 w-full text-border"
						role="img"
						aria-label="Network throughput during this session"
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
				</div>
			</section>

			<section
				class="mt-5 overflow-hidden rounded-xl border border-border"
				aria-labelledby="services-title"
			>
				<header
					class="flex flex-col gap-2 px-5 py-4 sm:flex-row sm:items-center sm:justify-between"
				>
					<div>
						<h2 id="services-title" class="text-[15px] font-semibold text-foreground">
							Services and containers
						</h2>
						<p class="mt-0.5 text-xs text-muted-foreground">
							Only active deployments contribute metrics; every service remains visible.
						</p>
					</div>
					<Badge
						tone={snapshot.summary.degraded_services ? 'warning' : 'success'}
						variant="outline"
					>
						{snapshot.summary.degraded_services
							? `${snapshot.summary.degraded_services} need attention`
							: 'All healthy'}
					</Badge>
				</header>
				<div class="overflow-x-auto border-t border-border">
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
								<tr class="border-t border-border first:border-t-0">
									<td class="max-w-[260px] px-5 py-3">
										<a
											href="/services/{service.service_id}"
											class="block truncate font-medium text-foreground hover:underline"
											>{service.name}</a
										>
										{#if service.error}<p
												class="mt-1 truncate text-xs text-destructive"
												title={service.error}
											>
												{service.error}
											</p>{/if}
									</td>
									<td class="px-4 py-3 text-xs text-muted-foreground"
										>{service.environment_name || 'Unknown'}</td
									>
									<td class="px-4 py-3"
										><Badge tone={statusTone(service.status)}>{statusLabel(service.status)}</Badge
										></td
									>
									<td
										class="px-4 py-3 text-right font-mono text-[13px] text-foreground tabular-nums"
										>{service.container ? formatPercent(service.container.cpu_percent) : '—'}</td
									>
									<td
										class="px-4 py-3 text-right font-mono text-[13px] text-foreground tabular-nums"
										>{service.container
											? formatBytes(service.container.memory_used_bytes)
											: '—'}</td
									>
									<td
										class="px-5 py-3 text-right font-mono text-[13px] text-foreground tabular-nums"
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
