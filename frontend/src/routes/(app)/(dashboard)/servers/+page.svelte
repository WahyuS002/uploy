<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import type { PageData } from './$types';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import PublicKeyHelper from '$lib/components/app/PublicKeyHelper.svelte';
	import StatusBadge from '$lib/components/app/StatusBadge.svelte';
	import ServerConnectWizard from '$lib/components/app/ServerConnectWizard.svelte';
	import ServerMetricTrend from '$lib/components/app/ServerMetricTrend.svelte';
	import LogStream from '$lib/components/app/LogStream.svelte';
	import { ServerCreateController } from '$lib/components/app/server-create-form.svelte';
	import { formatDate } from '$lib/format-date';
	import { Button, EmptyState, Input, Tooltip, toast } from '$lib/components/ui';
	import { Dialog, DialogContent, DialogHeader, DialogTitle } from '$lib/components/ui/dialog';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ChevronRight, InformationCircle, Key, Plus, Server } from '@steeze-ui/heroicons';

	let { data }: { data: PageData } = $props();

	let isOwner = $derived(data.workspace?.role === 'owner');
	let servers = $derived(data.servers ?? []);
	let keysById = $derived(new Map((data.keys ?? []).map((k) => [k.id, k])));
	let serverHealth = $derived(data.serverHealth ?? {});

	let dialogOpen = $state(false);
	let expandedServerId = $state<string | null>(null);
	let expandedHealthId = $state<string | null>(null);
	let upgradingProxyId = $state<string | null>(null);
	let monitoringActionId = $state<string | null>(null);
	let monitoringServer = $state<(typeof servers)[number] | null>(null);
	let monitoringPrivateAddress = $state('');
	let monitoringPort = $state('9184');
	let monitoringRetentionDays = $state('7');
	let monitoringFQDN = $state('');
	let monitoringReaderToken = $state('');

	// The server whose proxy log is open, or null. Held whole rather than by id so
	// the dialog can name the machine without looking it back up.
	let logServer = $state<(typeof servers)[number] | null>(null);

	const serverController = new ServerCreateController({
		onSuccess: async () => {
			dialogOpen = false;
			await invalidateAll();
		}
	});

	function openCreate() {
		serverController.reset();
		dialogOpen = true;
	}

	function formatBytes(value: number) {
		if (!Number.isFinite(value) || value <= 0) return '0 B';
		const units = ['B', 'KB', 'MB', 'GB', 'TB'];
		const exponent = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
		return `${(value / 1024 ** exponent).toFixed(exponent === 0 ? 0 : 1)} ${units[exponent]}`;
	}

	function diskHealthClass(percent: number) {
		if (percent >= 90) return 'border-destructive/30 bg-destructive/5 text-destructive';
		if (percent >= 80) return 'border-warning/30 bg-warning-muted text-warning';
		return 'border-border/70 bg-muted/30 text-foreground';
	}

	function diskActivityPerSecond(points: components['schemas']['ServerObservabilityResponse'][]) {
		const rates = diskActivityRates(points);
		return rates.read + rates.write;
	}

	function diskActivityRates(points: components['schemas']['ServerObservabilityResponse'][]) {
		if (points.length < 2) return { read: 0, write: 0 };
		const current = points[points.length - 1];
		const previous = points[points.length - 2];
		const elapsed = Math.max(
			1,
			(new Date(current.sampled_at).getTime() - new Date(previous.sampled_at).getTime()) / 1000
		);
		return {
			read: Math.max(0, (current.disk_read_bytes_total - previous.disk_read_bytes_total) / elapsed),
			write: Math.max(
				0,
				(current.disk_write_bytes_total - previous.disk_write_bytes_total) / elapsed
			)
		};
	}

	function openMonitoring(server: (typeof servers)[number]) {
		monitoringServer = server;
		monitoringPrivateAddress = server.monitoring.private_address;
		monitoringPort = String(server.monitoring.port || 9184);
		monitoringRetentionDays = String(server.monitoring.retention_days || 7);
		monitoringFQDN = server.monitoring.fqdn ?? '';
		monitoringReaderToken = '';
	}

	async function saveMonitoring() {
		if (!monitoringServer || monitoringActionId) return;
		const privateAddress = monitoringPrivateAddress.trim();
		const port = Number(monitoringPort);
		const retentionDays = Number(monitoringRetentionDays);
		const readerToken = monitoringReaderToken.trim();
		if (!privateAddress) {
			toast.error({
				title: 'Private address required',
				description: 'Use the address reachable from Uploy.'
			});
			return;
		}
		const readerTokenRequired =
			!monitoringServer.monitoring.enabled && !monitoringServer.monitoring.cleanup_at;
		if (readerTokenRequired && (readerToken.length < 32 || readerToken.length > 512)) {
			toast.error({
				title: 'Reader token required',
				description: 'Use 32 to 512 characters. It is shown only while you enter it.'
			});
			return;
		}
		if (!Number.isInteger(port) || port < 1 || port > 65535) {
			toast.error({ title: 'Invalid port', description: 'Use a port from 1 to 65535.' });
			return;
		}
		if (!Number.isInteger(retentionDays) || retentionDays < 1 || retentionDays > 30) {
			toast.error({ title: 'Invalid retention', description: 'Use 1 to 30 days.' });
			return;
		}

		monitoringActionId = monitoringServer.id;
		const toastId = `monitoring-save-${monitoringServer.id}`;
		toast.neutral({
			id: toastId,
			title: 'Provisioning monitoring',
			description: 'Installing the edge agent on the server.',
			icon: { kind: 'spinner' },
			dismissible: false
		});
		try {
			const { error } = await api.POST('/api/servers/{id}/monitoring', {
				params: { path: { id: monitoringServer.id } },
				body: {
					private_address: privateAddress,
					port,
					retention_days: retentionDays,
					fqdn: monitoringFQDN.trim(),
					...(readerToken ? { reader_token: readerToken } : {})
				}
			});
			if (error) {
				toast.error({
					id: toastId,
					title: 'Monitoring setup failed',
					description: (error as components['schemas']['ErrorResponse']).error
				});
				return;
			}
			monitoringServer = null;
			await invalidateAll();
			toast.success({
				id: toastId,
				title: 'Monitoring ready',
				description: 'Uploy now reads live and retained metrics through the edge agent.'
			});
		} catch {
			toast.error({
				id: toastId,
				title: 'Monitoring setup failed',
				description: 'Network error. Check the server connection, then retry.'
			});
		} finally {
			monitoringActionId = null;
		}
	}

	async function disableMonitoring(server: (typeof servers)[number]) {
		if (monitoringActionId) return;
		monitoringActionId = server.id;
		const toastId = `monitoring-disable-${server.id}`;
		try {
			const { error } = await api.DELETE('/api/servers/{id}/monitoring', {
				params: { path: { id: server.id } }
			});
			if (error) {
				toast.error({
					id: toastId,
					title: 'Monitoring disable failed',
					description: (error as components['schemas']['ErrorResponse']).error
				});
				return;
			}
			await invalidateAll();
			toast.success({
				id: toastId,
				title: 'Monitoring disabled',
				description: 'History remains on this server for seven days unless deleted now.'
			});
		} catch {
			toast.error({
				id: toastId,
				title: 'Monitoring disable failed',
				description: 'Network error.'
			});
		} finally {
			monitoringActionId = null;
		}
	}

	async function purgeMonitoringHistory(server: (typeof servers)[number]) {
		if (
			!window.confirm(
				`Delete all retained monitoring history on ${server.name}? This cannot be undone.`
			)
		)
			return;
		if (monitoringActionId) return;
		monitoringActionId = server.id;
		const toastId = `monitoring-purge-${server.id}`;
		try {
			const { error } = await api.DELETE('/api/servers/{id}/monitoring/history', {
				params: { path: { id: server.id } }
			});
			if (error) {
				toast.error({
					id: toastId,
					title: 'History deletion failed',
					description: (error as components['schemas']['ErrorResponse']).error
				});
				return;
			}
			await invalidateAll();
			toast.success({
				id: toastId,
				title: 'History deleted',
				description: 'Retained metrics are gone.'
			});
		} catch {
			toast.error({ id: toastId, title: 'History deletion failed', description: 'Network error.' });
		} finally {
			monitoringActionId = null;
		}
	}

	async function upgradeProxy(server: (typeof servers)[number]) {
		if (upgradingProxyId) return;
		upgradingProxyId = server.id;
		const toastId = `proxy-upgrade-${server.id}`;
		toast.neutral({
			id: toastId,
			title: 'Upgrading proxy',
			description: `${server.name} stays online while Traefik is reconciled.`,
			icon: { kind: 'spinner' },
			dismissible: false
		});
		try {
			const { error } = await api.POST('/api/servers/{id}/proxy', {
				params: { path: { id: server.id } }
			});
			if (error) {
				const message = (error as components['schemas']['ErrorResponse']).error;
				toast.error({ id: toastId, title: 'Proxy upgrade failed', description: message });
				return;
			}
			await invalidateAll();
			toast.success({
				id: toastId,
				title: 'Proxy ready',
				description: `${server.name} now supports rolling deployments.`,
				duration: 5000
			});
		} catch {
			toast.error({
				id: toastId,
				title: 'Proxy upgrade failed',
				description: 'Network error. Check the server connection, then retry.'
			});
		} finally {
			upgradingProxyId = null;
		}
	}
</script>

<svelte:head>
	<title>Servers · Uploy</title>
</svelte:head>

<section class="flex flex-1 flex-col">
	<div class="mb-5 flex items-center justify-between gap-4">
		<div class="min-w-0">
			<h1 class="text-xl font-semibold tracking-tight text-foreground">All servers</h1>
			{#if servers.length > 0}
				<p class="mt-1 text-sm text-muted-foreground">
					{servers.length}
					{servers.length === 1 ? 'server' : 'servers'} reachable over SSH.
				</p>
			{/if}
		</div>
		{#if isOwner}
			<Button variant="primary" size="sm" onclick={openCreate}>
				<Icon src={Plus} theme="outline" class="h-3.5 w-3.5" />
				Add server
			</Button>
		{/if}
	</div>

	<div class="flex flex-1 flex-col">
		{#if servers.length > 0}
			<div class="overflow-x-auto">
				<table class="w-full min-w-[56rem] text-left text-sm">
					<thead>
						<tr class="border-b border-border text-xs text-muted-foreground">
							<th scope="col" class="pb-2 font-medium">Server</th>
							<th scope="col" class="pb-2 font-medium">SSH Access</th>
							<th scope="col" class="pb-2 font-medium">Proxy Status</th>
							<th scope="col" class="pb-2 font-medium">Monitoring</th>
							<th scope="col" class="pb-2 font-medium">Created</th>
						</tr>
					</thead>
					<tbody>
						{#each servers as server (server.id)}
							{@const key = keysById.get(server.ssh_key_id)}
							{@const expanded = expandedServerId === server.id}
							{@const health = serverHealth[server.id]}
							<tr class="border-b border-border">
								<td class="py-2.5 align-top">
									<p class="font-medium text-foreground">{server.name}</p>
									<p class="mt-0.5 font-mono text-xs text-muted-foreground">
										{server.host}:{server.port}
									</p>
								</td>
								<td class="py-2.5 align-top">
									<p class="font-mono text-xs text-muted-foreground">{server.ssh_user}</p>
									{#if key?.public_key}
										<button
											type="button"
											id="public-key-toggle-{server.id}"
											aria-expanded={expanded}
											aria-controls="public-key-{server.id}"
											onclick={() => (expandedServerId = expanded ? null : server.id)}
											class="mt-1 inline-flex cursor-pointer items-center gap-1 rounded text-xs text-foreground underline decoration-border underline-offset-2 transition-colors hover:decoration-foreground focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
										>
											<Icon src={Key} theme="outline" class="h-3 w-3 text-muted-foreground" />
											{key.name}
										</button>
									{:else}
										<p class="mt-1 inline-flex items-center gap-1 text-xs text-muted-foreground">
											<Icon src={Key} theme="outline" class="h-3 w-3" />
											{key?.name ?? 'Key unavailable'}
										</p>
									{/if}
								</td>
								<td class="py-2.5 align-top">
									<div class="flex items-center gap-1.5">
										<StatusBadge status={server.proxy_status} />
										{#if server.proxy_last_error}
											<Tooltip
												text={server.proxy_last_error}
												ariaLabel="Proxy error: {server.proxy_last_error}"
												triggerClass="grid h-5 w-5 place-items-center rounded text-destructive transition-colors hover:text-destructive/80 focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
											>
												{#snippet children()}
													<Icon src={InformationCircle} theme="outline" class="h-3.5 w-3.5" />
												{/snippet}
											</Tooltip>
										{/if}
									</div>
									<!-- Under the status, not in a column of its own: the log is what you
									     reach for to find out what the badge means, and a proxy that never
									     came up is exactly when it is worth reading. -->
									<div class="mt-1 flex flex-wrap items-center gap-2">
										<button
											type="button"
											onclick={() => (logServer = server)}
											class="cursor-pointer rounded text-xs text-foreground underline decoration-border underline-offset-2 transition-colors hover:decoration-foreground focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
										>
											Proxy logs
										</button>
										{#if isOwner && server.proxy_status !== 'ready'}
											<Button
												variant="secondary"
												size="xs"
												loading={upgradingProxyId === server.id}
												disabled={upgradingProxyId !== null}
												onclick={() => upgradeProxy(server)}
											>
												{server.proxy_status === 'not_configured'
													? 'Install proxy'
													: 'Upgrade proxy'}
											</Button>
										{/if}
									</div>
								</td>
								<td class="py-2.5 align-top">
									<div class="flex items-center gap-1.5">
										<StatusBadge status={server.monitoring.status} />
										{#if server.monitoring.last_error}
											<Tooltip
												text={server.monitoring.last_error}
												ariaLabel="Monitoring error: {server.monitoring.last_error}"
												triggerClass="grid h-5 w-5 place-items-center rounded text-destructive transition-colors hover:text-destructive/80 focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
											>
												{#snippet children()}
													<Icon src={InformationCircle} theme="outline" class="h-3.5 w-3.5" />
												{/snippet}
											</Tooltip>
										{/if}
									</div>
									{#if server.monitoring.enabled}
										<p class="mt-1 font-mono text-xs text-muted-foreground">
											{server.monitoring.fqdn ??
												`${server.monitoring.private_address}:${server.monitoring.port}`}
										</p>
									{/if}
									{#if server.monitoring.cleanup_at}
										<p class="mt-1 text-xs text-muted-foreground">
											History expires {formatDate(server.monitoring.cleanup_at ?? '')}
										</p>
									{/if}
									{#if health?.latest}
										<div class="mt-3 space-y-1.5 text-xs">
											<div
												class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 rounded border px-2 py-1.5 {diskHealthClass(
													health.latest.disk_used_percent
												)}"
											>
												<span class="font-medium">Disk</span>
												<span class="tabular-nums">
													{formatBytes(health.latest.disk_used_bytes)} / {formatBytes(
														health.latest.disk_total_bytes
													)}
													({health.latest.disk_used_percent.toFixed(0)}%)
												</span>
											</div>
											<div class="flex items-center justify-between gap-3 text-muted-foreground">
												<span>Disk I/O</span>
												<span class="tabular-nums"
													>R {formatBytes(diskActivityRates(health.history).read)}/s / W
													{formatBytes(diskActivityRates(health.history).write)}/s</span
												>
											</div>
											<div class="flex items-center justify-between gap-3 text-muted-foreground">
												<span>Load</span>
												<span class="tabular-nums"
													>{health.latest.load_1.toFixed(1)} / {health.latest.load_5.toFixed(1)} /
													{health.latest.load_15.toFixed(1)}</span
												>
											</div>
											<div class="flex items-center justify-between gap-3 text-muted-foreground">
												<span>Swap</span>
												<span class="tabular-nums"
													>{formatBytes(health.latest.swap_used_bytes)} / {formatBytes(
														health.latest.swap_total_bytes
													)}</span
												>
											</div>
										</div>
										{#if health.history.length > 1}
											<button
												type="button"
												class="mt-2 cursor-pointer rounded text-xs text-foreground underline decoration-border underline-offset-2 transition-colors hover:decoration-foreground focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
												onclick={() =>
													(expandedHealthId = expandedHealthId === server.id ? null : server.id)}
												aria-expanded={expandedHealthId === server.id}
											>
												{expandedHealthId === server.id ? 'Hide trends' : '24h health trends'}
											</button>
										{/if}
									{:else if server.monitoring.enabled}
										<p class="mt-2 text-xs text-muted-foreground">Health metrics unavailable.</p>
									{/if}
									{#if isOwner}
										<div class="mt-1 flex flex-wrap items-center gap-2">
											<Button
												variant="secondary"
												size="xs"
												loading={monitoringActionId === server.id}
												disabled={monitoringActionId !== null}
												onclick={() => openMonitoring(server)}
											>
												{server.monitoring.enabled ? 'Configure' : 'Enable'}
											</Button>
											{#if server.monitoring.enabled}
												<Button
													variant="ghost"
													size="xs"
													disabled={monitoringActionId !== null}
													onclick={() => disableMonitoring(server)}
												>
													Disable
												</Button>
											{:else if server.monitoring.cleanup_at}
												<Button
													variant="ghost"
													size="xs"
													disabled={monitoringActionId !== null}
													onclick={() => purgeMonitoringHistory(server)}
												>
													Delete history
												</Button>
											{/if}
										</div>
									{/if}
								</td>
								<td class="py-2.5 align-top text-muted-foreground">
									{formatDate(server.created_at)}
								</td>
							</tr>
							{#if expanded && key?.public_key}
								<tr class="border-b border-border">
									<td colspan="5" class="pt-1 pb-3">
										<div id="public-key-{server.id}" class="max-w-2xl">
											<PublicKeyHelper
												publicKey={key.public_key}
												description="Add to ~/.ssh/authorized_keys for {server.ssh_user} on {server.host}."
											/>
										</div>
									</td>
								</tr>
							{/if}
							{#if expandedHealthId === server.id && health?.history.length}
								<tr class="border-b border-border bg-muted/20">
									<td colspan="5" class="pt-1 pb-4">
										<div class="grid gap-4 px-1 pt-3 sm:grid-cols-2 xl:grid-cols-4">
											<ServerMetricTrend
												points={health?.history ?? []}
												metric="disk"
												label="Disk usage"
												value={`${health?.latest?.disk_used_percent.toFixed(0) ?? '-'}%`}
											/>
											<ServerMetricTrend
												points={health?.history ?? []}
												metric="io"
												label="Disk activity"
												value={`${formatBytes(diskActivityPerSecond(health?.history ?? []))}/s`}
											/>
											<ServerMetricTrend
												points={health?.history ?? []}
												metric="load"
												label="Load average (1m)"
												value={health?.latest?.load_1.toFixed(1) ?? '-'}
											/>
											<ServerMetricTrend
												points={health?.history ?? []}
												metric="swap"
												label="Swap used"
												value={formatBytes(health?.latest?.swap_used_bytes ?? 0)}
											/>
										</div>
									</td>
								</tr>
							{/if}
						{/each}
					</tbody>
				</table>
			</div>
		{:else}
			<EmptyState
				variant="canvas"
				icon={Server}
				title="No servers connected yet"
				description={isOwner
					? 'Connect your first server to start deploying services to your own infrastructure. You can generate an SSH key along the way.'
					: 'Ask a workspace owner to connect a server before deploying.'}
			>
				{#snippet actions()}
					{#if isOwner}
						<Button variant="primary" size="sm" onclick={openCreate}>
							<Icon src={Plus} theme="outline" class="h-3.5 w-3.5" />
							Add server
						</Button>
					{/if}
				{/snippet}
			</EmptyState>
		{/if}
	</div>
</section>

<Dialog bind:open={dialogOpen}>
	<DialogContent class="w-[min(92vw,32rem)] max-w-none overflow-hidden">
		<DialogHeader class="border-b border-border px-5 pt-4 pr-12 pb-3">
			<DialogTitle class="flex items-center gap-2 text-sm">
				<span class="font-medium text-muted-foreground">Servers</span>
				<Icon src={ChevronRight} theme="outline" class="h-3.5 w-3.5 text-muted-foreground" />
				<span
					class="grid h-6 w-6 place-content-center rounded-full border border-border bg-muted"
					aria-hidden="true"
				>
					<Icon src={Server} theme="outline" class="h-3.5 w-3.5 text-muted-foreground" />
				</span>
				New server
			</DialogTitle>
		</DialogHeader>
		<ServerConnectWizard
			controller={serverController}
			bodyClass="max-h-[min(65vh,32rem)] overflow-y-auto px-5 pt-4 pb-5"
			actionsClass="rounded-b-xl border-t border-border px-5 py-3"
		/>
	</DialogContent>
</Dialog>

<Dialog
	open={monitoringServer !== null}
	onOpenChange={(open) => !open && (monitoringServer = null)}
>
	<DialogContent class="w-[min(92vw,34rem)] max-w-none overflow-hidden">
		<DialogHeader class="border-b border-border px-5 pt-4 pr-12 pb-3">
			<DialogTitle class="flex min-w-0 items-center gap-2 text-sm">
				<span class="flex-none font-medium text-muted-foreground">Monitoring</span>
				<Icon
					src={ChevronRight}
					theme="outline"
					class="h-3.5 w-3.5 flex-none text-muted-foreground"
				/>
				<span class="truncate">{monitoringServer?.name}</span>
			</DialogTitle>
		</DialogHeader>
		<form
			class="space-y-4 px-5 py-4"
			onsubmit={(event) => {
				event.preventDefault();
				void saveMonitoring();
			}}
		>
			<p class="text-sm leading-6 text-muted-foreground">
				Stores container metrics locally on this server. Uploy reads the private HTTP API; a public
				FQDN is optional for external scrapers.
			</p>
			<label class="flex flex-col gap-1.5">
				<span class="text-sm font-medium text-foreground">Private address</span>
				<Input bind:value={monitoringPrivateAddress} placeholder="10.0.0.4" autocomplete="off" />
			</label>
			<div class="grid grid-cols-2 gap-3">
				<label class="flex flex-col gap-1.5">
					<span class="text-sm font-medium text-foreground">Port</span>
					<Input bind:value={monitoringPort} inputmode="numeric" />
				</label>
				<label class="flex flex-col gap-1.5">
					<span class="text-sm font-medium text-foreground">Retention days</span>
					<Input bind:value={monitoringRetentionDays} inputmode="numeric" />
				</label>
			</div>
			<label class="flex flex-col gap-1.5">
				<span class="text-sm font-medium text-foreground"
					>Public FQDN <span class="font-normal text-muted-foreground">(optional)</span></span
				>
				<Input bind:value={monitoringFQDN} placeholder="metrics.example.com" autocomplete="off" />
			</label>
			<label class="flex flex-col gap-1.5">
				<span class="text-sm font-medium text-foreground"
					>Reader token {#if !monitoringServer?.monitoring.enabled && !monitoringServer?.monitoring.cleanup_at}<span
							class="text-destructive">required</span
						>{:else}<span class="font-normal text-muted-foreground"
							>(leave blank to keep current)</span
						>{/if}</span
				>
				<Input
					bind:value={monitoringReaderToken}
					type="password"
					minlength={32}
					maxlength={512}
					autocomplete="new-password"
				/>
				<p class="text-xs leading-5 text-muted-foreground">
					This token authorizes `/metrics` and read-only JSON endpoints. Uploy never displays it
					again.
				</p>
			</label>
			<div class="flex justify-end gap-2 border-t border-border pt-4">
				<Button type="button" variant="ghost" size="sm" onclick={() => (monitoringServer = null)}
					>Cancel</Button
				>
				<Button type="submit" size="sm" loading={monitoringActionId === monitoringServer?.id}
					>Save monitoring</Button
				>
			</div>
		</form>
	</DialogContent>
</Dialog>

<!-- A dialog rather than a page of its own: there is no server detail view to
     hang it off, and the log is something you glance at from the row you were
     already reading. -->
<Dialog open={logServer !== null} onOpenChange={(open) => !open && (logServer = null)}>
	<DialogContent class="w-[min(92vw,56rem)] max-w-none overflow-hidden">
		<DialogHeader class="border-b border-border px-5 pt-4 pr-12 pb-3">
			<DialogTitle class="flex min-w-0 items-center gap-2 text-sm">
				<span class="flex-none font-medium text-muted-foreground">Proxy logs</span>
				<Icon
					src={ChevronRight}
					theme="outline"
					class="h-3.5 w-3.5 flex-none text-muted-foreground"
				/>
				<span class="truncate">{logServer?.name}</span>
			</DialogTitle>
		</DialogHeader>
		<!-- A fixed height, not a max: the panel scrolls its own output, and a box
		     that grows as lines arrive would move the whole dialog under the cursor. -->
		<div class="h-[min(66vh,34rem)] p-5">
			<!-- Keyed so opening a second server's log tears the first stream down
			     instead of leaving one SSH session per row you clicked. -->
			{#if logServer}
				{#key logServer.id}
					<LogStream
						endpoint="/api/servers/{logServer.id}/proxy-logs"
						subject="proxy"
						class="h-full"
					/>
				{/key}
			{/if}
		</div>
	</DialogContent>
</Dialog>
