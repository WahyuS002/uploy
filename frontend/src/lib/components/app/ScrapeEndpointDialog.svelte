<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import CopyButton from './CopyButton.svelte';
	import { Alert, Button, CodeBlock, toast } from '$lib/components/ui';
	import { Dialog, DialogContent, DialogHeader, DialogTitle } from '$lib/components/ui/dialog';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ChevronRight } from '@steeze-ui/heroicons';

	type ErrorResponse = components['schemas']['ErrorResponse'];
	type Credentials = components['schemas']['ServerMonitoringCredentialsResponse'];

	type Props = {
		/** The server whose credentials to show, or null when the dialog is closed. */
		server: { id: string; name: string; monitoring: { port: number } } | null;
		onClose: () => void;
	};

	let { server, onClose }: Props = $props();

	let credentials = $state<Credentials | null>(null);
	let loadError = $state('');
	let revealed = $state(false);
	let rotating = $state(false);

	// Fetched on open rather than carried in the servers list: the token is only
	// meaningful to someone who deliberately came looking for it, so it stays out of
	// every other response that page renders.
	$effect(() => {
		const target = server;
		let cancelled = false;
		credentials = null;
		loadError = '';
		revealed = false;
		if (!target) return;
		void (async () => {
			try {
				const { data, error } = await api.GET('/api/servers/{id}/monitoring/credentials', {
					params: { path: { id: target.id } }
				});
				if (cancelled) return;
				if (error || !data) {
					loadError = (error as ErrorResponse)?.error ?? 'Could not load the scrape credentials.';
					return;
				}
				credentials = data;
			} catch {
				if (!cancelled) loadError = 'Could not load the scrape credentials.';
			}
		})();
		return () => {
			cancelled = true;
		};
	});

	let token = $derived(credentials?.reader_token ?? '');
	let maskedToken = $derived(token ? `${token.slice(0, 6)}${'•'.repeat(26)}` : '');
	let scrapeConfig = $derived(
		credentials?.metrics_url
			? [
					'scrape_configs:',
					`  - job_name: uploy-${server?.name ?? 'server'}`,
					'    scheme: https',
					'    metrics_path: /metrics',
					'    authorization:',
					`      credentials: ${token}`,
					'    static_configs:',
					`      - targets: ['${new URL(credentials.metrics_url).host}']`
				].join('\n')
			: ''
	);

	async function rotate() {
		if (!server || rotating) return;
		if (
			!confirm(
				`Replace the reader token on ${server.name}? Any external collector using the current token stops receiving metrics until you update it.`
			)
		) {
			return;
		}
		rotating = true;
		const toastId = `monitoring-rotate-${server.id}`;
		toast.neutral({
			id: toastId,
			title: 'Rotating reader token',
			description: 'Restarting the edge agent with the replacement token.',
			icon: { kind: 'spinner' },
			dismissible: false
		});
		try {
			const { data, error } = await api.POST('/api/servers/{id}/monitoring/reader-token', {
				params: { path: { id: server.id } }
			});
			if (error || !data) {
				toast.error({
					id: toastId,
					title: 'Rotation failed',
					description: (error as ErrorResponse)?.error ?? 'The agent kept its current token.'
				});
				return;
			}
			credentials = data;
			revealed = true;
			toast.success({
				id: toastId,
				title: 'Reader token replaced',
				description: 'Update your collector with the new token.'
			});
		} catch {
			toast.error({
				id: toastId,
				title: 'Rotation failed',
				description: 'Network error. Check the server connection, then retry.'
			});
		} finally {
			rotating = false;
		}
	}
</script>

<Dialog open={server !== null} onOpenChange={(open) => !open && onClose()}>
	<DialogContent class="w-[min(92vw,38rem)] max-w-none overflow-hidden">
		<DialogHeader class="border-b border-border px-5 pt-4 pr-12 pb-3">
			<DialogTitle class="flex min-w-0 items-center gap-2 text-sm">
				<span class="flex-none font-medium text-muted-foreground">Scrape endpoint</span>
				<Icon
					src={ChevronRight}
					theme="outline"
					class="h-3.5 w-3.5 flex-none text-muted-foreground"
				/>
				<span class="truncate">{server?.name ?? '…'}</span>
			</DialogTitle>
		</DialogHeader>
		{#if loadError}
			<p class="px-5 py-8 text-center text-sm text-muted-foreground">{loadError}</p>
		{:else if !credentials}
			<div class="space-y-4 px-5 py-4" aria-busy="true">
				<div class="h-4 w-2/3 animate-pulse rounded bg-muted"></div>
				<div class="h-9 animate-pulse rounded-md bg-muted"></div>
				<div class="h-24 animate-pulse rounded-md bg-muted"></div>
			</div>
		{:else}
			<div class="max-h-[70vh] space-y-4 overflow-y-auto px-5 py-4">
				<p class="text-sm leading-6 text-muted-foreground">
					Uploy's own charts need none of this — it reads the agent over SSH. These are for pointing
					a Prometheus-compatible collector at the same metrics.
				</p>

				<div class="flex flex-col gap-1.5">
					<span class="text-sm font-medium text-foreground">Endpoint</span>
					{#if credentials.metrics_url}
						<div class="flex items-center gap-2">
							<code
								class="min-w-0 flex-1 truncate rounded-md bg-muted px-2.5 py-2 font-mono text-xs"
							>
								{credentials.metrics_url}
							</code>
							<CopyButton text={credentials.metrics_url} defaultLabel="Copy" />
						</div>
					{:else}
						<Alert tone="info">
							The agent is only published on
							<code class="font-mono">127.0.0.1:{server?.monitoring.port}</code>, so nothing outside
							the machine can reach it. Add a public FQDN under Configure → Advanced to expose it
							through HTTPS, or scrape it from a collector running on the same host.
						</Alert>
					{/if}
				</div>

				<div class="flex flex-col gap-1.5">
					<div class="flex items-center justify-between gap-2">
						<span class="text-sm font-medium text-foreground">Reader token</span>
						<button
							type="button"
							class="cursor-pointer text-xs text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
							onclick={() => (revealed = !revealed)}
						>
							{revealed ? 'Hide' : 'Reveal'}
						</button>
					</div>
					<div class="flex items-center gap-2">
						<code
							class="min-w-0 flex-1 truncate rounded-md bg-muted px-2.5 py-2 font-mono text-xs"
							aria-label="Reader token"
						>
							{revealed ? token : maskedToken}
						</code>
						<CopyButton text={token} defaultLabel="Copy" />
					</div>
					<p class="text-xs leading-5 text-muted-foreground">
						Read-only: it authorizes <code class="font-mono">/metrics</code> and the JSON read endpoints,
						and nothing else. Rotate it if it leaks — collectors holding the old one start getting 401s.
					</p>
				</div>

				{#if scrapeConfig}
					<div class="flex flex-col gap-1.5">
						<div class="flex items-center justify-between gap-2">
							<span class="text-sm font-medium text-foreground">Prometheus scrape config</span>
							<CopyButton text={scrapeConfig} defaultLabel="Copy config" variant="ghost" />
						</div>
						<CodeBlock code={revealed ? scrapeConfig : scrapeConfig.replace(token, maskedToken)} />
					</div>
				{/if}

				<div class="flex justify-end gap-2 border-t border-border pt-4">
					<Button
						type="button"
						variant="ghost"
						size="sm"
						loading={rotating}
						onclick={() => void rotate()}
					>
						Rotate token
					</Button>
					<Button type="button" size="sm" onclick={onClose}>Done</Button>
				</div>
			</div>
		{/if}
	</DialogContent>
</Dialog>
