<script lang="ts">
	import Badge from '$lib/components/ui/Badge.svelte';
	import { Icon } from '@steeze-ui/svelte-icon';
	import {
		ArrowTrendingDown,
		ArrowTrendingUp,
		CircleStack,
		CpuChip,
		Server,
		Signal
	} from '@steeze-ui/heroicons';

	const metrics = [
		{
			label: 'CPU usage',
			value: '38%',
			detail: '4 vCPU',
			trend: '6.2%',
			trendLabel: 'higher than previous hour',
			trendIcon: ArrowTrendingUp,
			icon: CpuChip,
			class: 'text-info'
		},
		{
			label: 'Memory',
			value: '5.8 GB',
			detail: 'of 8 GB',
			trend: '2.1%',
			trendLabel: 'lower than previous hour',
			trendIcon: ArrowTrendingDown,
			icon: CircleStack,
			class: 'text-success'
		},
		{
			label: 'Disk usage',
			value: '42%',
			detail: '34 of 80 GB',
			trend: '1.4 GB',
			trendLabel: 'written in the last hour',
			trendIcon: ArrowTrendingUp,
			icon: Server,
			class: 'text-warning'
		},
		{
			label: 'Network',
			value: '18.6 MB/s',
			detail: '12.4 in · 6.2 out',
			trend: '8.4%',
			trendLabel: 'higher than previous hour',
			trendIcon: ArrowTrendingUp,
			icon: Signal,
			class: 'text-primary-deep'
		}
	];

	const processes = [
		{
			name: 'docker',
			detail: 'Container runtime',
			cpu: '12.8%',
			memory: '1.4 GB',
			state: 'Healthy'
		},
		{ name: 'postgres', detail: 'Database', cpu: '8.4%', memory: '2.1 GB', state: 'Healthy' },
		{ name: 'traefik', detail: 'Edge proxy', cpu: '4.9%', memory: '328 MB', state: 'Healthy' },
		{ name: 'node', detail: 'Application', cpu: '3.7%', memory: '612 MB', state: 'Healthy' }
	];
</script>

<svelte:head>
	<title>VM observability · Uploy</title>
</svelte:head>

<div class="min-h-0 flex-1 overflow-y-auto rounded-xl border border-border bg-card">
	<div class="mx-auto w-full max-w-6xl px-5 py-8 sm:px-8 sm:py-10">
		<header class="mb-7 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
			<div class="max-w-2xl">
				<h1 class="text-2xl font-semibold tracking-[-0.02em] text-foreground">VM observability</h1>
				<p class="mt-2 text-sm leading-relaxed text-muted-foreground">
					Inspect resource pressure, throughput, and the workloads consuming this virtual machine.
				</p>
			</div>
			<div class="flex flex-wrap items-center gap-2">
				<Badge tone="warning">Preview data</Badge>
				<span class="rounded-md border border-border px-2.5 py-1 text-xs text-muted-foreground">
					Last 60 minutes
				</span>
			</div>
		</header>

		<section class="overflow-hidden rounded-xl border border-border" aria-label="VM overview">
			<div
				class="flex flex-col gap-3 border-b border-border px-5 py-4 sm:flex-row sm:items-center sm:justify-between"
			>
				<div class="flex min-w-0 items-center gap-3">
					<span
						class="grid h-9 w-9 flex-none place-content-center rounded-lg bg-muted text-foreground"
					>
						<Icon src={Server} theme="outline" class="h-4.5 w-4.5" />
					</span>
					<div class="min-w-0">
						<h2 class="truncate text-[15px] font-semibold text-foreground">production-vm-01</h2>
						<p class="mt-0.5 truncate font-mono text-xs text-muted-foreground">
							203.0.113.24 · Ubuntu 24.04
						</p>
					</div>
				</div>
				<div class="flex items-center gap-2 text-xs text-success">
					<span class="h-1.5 w-1.5 rounded-full bg-current"></span>
					Healthy · 24d uptime
				</div>
			</div>

			<div class="grid sm:grid-cols-2 xl:grid-cols-4">
				{#each metrics as metric, index (metric.label)}
					<div
						class="px-5 py-5 {index < metrics.length - 1
							? 'border-b border-border sm:border-r xl:border-b-0'
							: ''} {index === 1 ? 'sm:border-r-0 xl:border-r' : ''} {index === 2
							? 'sm:border-b-0'
							: ''}"
					>
						<div class="flex items-center justify-between gap-3">
							<p class="text-xs font-medium text-muted-foreground">{metric.label}</p>
							<Icon src={metric.icon} theme="outline" class={`h-4 w-4 ${metric.class}`} />
						</div>
						<div class="mt-3 flex items-baseline gap-2">
							<p class="text-2xl font-semibold tracking-[-0.02em] text-foreground">
								{metric.value}
							</p>
							<span class="text-xs text-muted-foreground">{metric.detail}</span>
						</div>
						<p class="mt-2 flex items-center gap-1 text-xs text-muted-foreground">
							<Icon src={metric.trendIcon} theme="outline" class="h-3.5 w-3.5" />
							<span class="font-medium text-foreground">{metric.trend}</span>
							{metric.trendLabel}
						</p>
					</div>
				{/each}
			</div>
		</section>

		<section
			class="mt-5 overflow-hidden rounded-xl border border-border"
			aria-labelledby="usage-title"
		>
			<header class="flex items-center justify-between gap-3 px-5 py-4">
				<div>
					<h2 id="usage-title" class="text-[15px] font-semibold text-foreground">Resource usage</h2>
					<p class="mt-0.5 text-xs text-muted-foreground">Illustrative one-minute samples</p>
				</div>
				<div class="flex items-center gap-4 text-xs">
					<span class="flex items-center gap-1.5 text-muted-foreground">
						<span class="h-1.5 w-1.5 rounded-full bg-info"></span>CPU
					</span>
					<span class="flex items-center gap-1.5 text-muted-foreground">
						<span class="h-1.5 w-1.5 rounded-full bg-success"></span>Memory
					</span>
				</div>
			</header>

			<div class="grid border-t border-border lg:grid-cols-[minmax(0,1.8fr)_minmax(260px,1fr)]">
				<div class="min-w-0 px-4 py-5 sm:px-5">
					<div class="h-64 w-full text-border">
						<svg
							viewBox="0 0 720 240"
							class="h-full w-full overflow-visible"
							role="img"
							aria-label="Illustrative CPU and memory usage over the last sixty minutes"
							preserveAspectRatio="none"
						>
							<path
								d="M0 24H720M0 72H720M0 120H720M0 168H720M0 216H720"
								stroke="currentColor"
								stroke-width="1"
							/>
							<path
								d="M0 216V0M144 216V0M288 216V0M432 216V0M576 216V0M720 216V0"
								stroke="currentColor"
								stroke-width="1"
							/>
							<path
								d="M0 166 C36 151 58 160 90 142 S148 112 180 126 S240 170 270 145 S326 96 360 108 S418 146 450 126 S510 84 540 98 S596 138 630 116 S684 74 720 88"
								class="text-info"
								fill="none"
								stroke="currentColor"
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="3"
							/>
							<path
								d="M0 126 C46 122 62 116 90 118 S142 132 180 120 S236 104 270 110 S328 130 360 122 S414 102 450 106 S514 124 540 114 S596 96 630 102 S686 112 720 98"
								class="text-success"
								fill="none"
								stroke="currentColor"
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="3"
							/>
						</svg>
					</div>
					<div class="mt-3 flex justify-between text-[11px] text-muted-foreground tabular-nums">
						<span>60m ago</span><span>45m</span><span>30m</span><span>15m</span><span>Now</span>
					</div>
				</div>

				<div class="border-t border-border px-5 py-5 lg:border-t-0 lg:border-l">
					<h3 class="text-sm font-semibold text-foreground">Capacity</h3>
					<div class="mt-5 space-y-5">
						<div>
							<div class="flex items-center justify-between text-xs">
								<span class="text-muted-foreground">CPU headroom</span>
								<span class="font-medium text-foreground tabular-nums">62%</span>
							</div>
							<div class="mt-2 h-1.5 overflow-hidden rounded-full bg-muted">
								<div class="h-full w-[62%] rounded-full bg-info"></div>
							</div>
						</div>
						<div>
							<div class="flex items-center justify-between text-xs">
								<span class="text-muted-foreground">Memory headroom</span>
								<span class="font-medium text-foreground tabular-nums">28%</span>
							</div>
							<div class="mt-2 h-1.5 overflow-hidden rounded-full bg-muted">
								<div class="h-full w-[28%] rounded-full bg-success"></div>
							</div>
						</div>
						<div>
							<div class="flex items-center justify-between text-xs">
								<span class="text-muted-foreground">Disk headroom</span>
								<span class="font-medium text-foreground tabular-nums">58%</span>
							</div>
							<div class="mt-2 h-1.5 overflow-hidden rounded-full bg-muted">
								<div class="h-full w-[58%] rounded-full bg-warning"></div>
							</div>
						</div>
					</div>

					<div class="mt-6 border-t border-border pt-4">
						<p class="text-xs text-muted-foreground">Peak CPU</p>
						<p class="mt-1 text-lg font-semibold text-foreground tabular-nums">
							71% <span class="text-xs font-normal text-muted-foreground">at 14:42</span>
						</p>
					</div>
				</div>
			</div>
		</section>

		<section
			class="mt-5 overflow-hidden rounded-xl border border-border"
			aria-labelledby="process-title"
		>
			<header class="flex items-center justify-between gap-3 px-5 py-4">
				<div>
					<h2 id="process-title" class="text-[15px] font-semibold text-foreground">
						Top processes
					</h2>
					<p class="mt-0.5 text-xs text-muted-foreground">Example workloads sorted by CPU usage</p>
				</div>
				<Badge tone="success" variant="outline">4 healthy</Badge>
			</header>
			<div class="overflow-x-auto border-t border-border">
				<table class="w-full min-w-150 text-left text-sm">
					<thead class="bg-muted/40 text-xs text-muted-foreground">
						<tr>
							<th class="px-5 py-2.5 font-medium">Process</th>
							<th class="px-4 py-2.5 font-medium">CPU</th>
							<th class="px-4 py-2.5 font-medium">Memory</th>
							<th class="px-5 py-2.5 text-right font-medium">State</th>
						</tr>
					</thead>
					<tbody>
						{#each processes as process (process.name)}
							<tr class="border-t border-border first:border-t-0">
								<td class="px-5 py-3">
									<p class="font-mono text-[13px] font-medium text-foreground">{process.name}</p>
									<p class="mt-0.5 text-xs text-muted-foreground">{process.detail}</p>
								</td>
								<td class="px-4 py-3 font-mono text-[13px] text-foreground tabular-nums"
									>{process.cpu}</td
								>
								<td class="px-4 py-3 font-mono text-[13px] text-foreground tabular-nums"
									>{process.memory}</td
								>
								<td class="px-5 py-3 text-right">
									<span class="inline-flex items-center gap-1.5 text-xs text-success">
										<span class="h-1.5 w-1.5 rounded-full bg-current"></span>{process.state}
									</span>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</section>
	</div>
</div>
