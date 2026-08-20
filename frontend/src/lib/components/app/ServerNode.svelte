<script lang="ts">
	import type { components } from '$lib/api/v1';
	import Button from '$lib/components/ui/Button.svelte';
	import IconButton from '$lib/components/ui/IconButton.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import StatusBadge from '$lib/components/app/StatusBadge.svelte';
	import RadioList from '$lib/components/ui/RadioList.svelte';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { Plus, Server } from '@steeze-ui/heroicons';

	type ServerResponse = components['schemas']['ServerResponse'];

	type Props = {
		servers: ServerResponse[];
		value?: string;
		loading?: boolean;
		error?: string;
		canConnect?: boolean;
		onValueChange?: (id: string) => void;
		onConnect: () => void;
		onRetry?: () => void;
	};

	let {
		servers,
		value = '',
		loading = false,
		error = '',
		canConnect = false,
		onValueChange,
		onConnect,
		onRetry
	}: Props = $props();

	const headingId = $props.id();

	// Only surface proxy state the user can act on. 'not_configured' is the column
	// default every freshly connected server carries, and it only clears once
	// something with a domain deploys — so it flags healthy servers as broken and
	// points at no fix. Silence means fine; a badge means look at this.
	const actionableProxyStatuses = new Set(['port_conflict', 'degraded']);

	function proxyProblem(status: string): boolean {
		return actionableProxyStatuses.has(status);
	}
</script>

<section
	class="panel w-full overflow-hidden rounded-lg border border-border bg-card text-card-foreground"
	aria-labelledby={headingId}
>
	<div class="flex items-center justify-between gap-2 border-b border-border/70 px-3 py-2">
		<h2 id={headingId} class="text-sm font-medium text-foreground">Server</h2>
		{#if canConnect && servers.length > 0}
			<IconButton size="sm" onclick={onConnect} aria-label="Connect another server">
				<Icon src={Plus} theme="outline" class="h-3.5 w-3.5" />
			</IconButton>
		{/if}
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-6">
			<Spinner class="h-4 w-4 text-muted-foreground" />
		</div>
	{:else if error}
		<div class="flex flex-col items-start gap-2 px-3 py-3">
			<p class="text-sm text-destructive">{error}</p>
			{#if onRetry}
				<Button type="button" size="xs" variant="secondary" onclick={onRetry}>Retry</Button>
			{/if}
		</div>
	{:else if servers.length === 0}
		<!-- The snippet is passed only when it has something to render: an empty
		     actions block would still open a 20px gap under the title. -->
		<EmptyState
			icon={Server}
			title="No servers connected"
			description={canConnect ? undefined : 'Ask a workspace owner to connect one.'}
			actions={canConnect ? connect : undefined}
			class="px-4 py-8"
		/>
	{:else}
		<RadioList
			items={servers}
			{value}
			name={headingId}
			ariaLabelledby={headingId}
			onChange={(id) => onValueChange?.(id)}
			class="max-h-52"
		>
			{#snippet children(server)}
				<span class="min-w-0 flex-1" title={server.name}>
					<span class="block truncate text-sm text-foreground">{server.name}</span>
					<span class="block truncate font-mono text-[11px] text-muted-foreground">
						{server.host}:{server.port}
					</span>
				</span>
				{#if proxyProblem(server.proxy_status)}
					<StatusBadge status={server.proxy_status} />
				{/if}
			{/snippet}
		</RadioList>
	{/if}
</section>

{#snippet connect()}
	<Button type="button" size="xs" variant="secondary" onclick={onConnect}>
		<Icon src={Plus} theme="outline" class="h-3 w-3" />
		Connect a server
	</Button>
{/snippet}

<style>
	.panel {
		box-shadow: var(--shadow-panel);
	}
</style>
