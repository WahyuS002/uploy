<script lang="ts">
	import { Select } from 'bits-ui';
	import type { components } from '$lib/api/v1';
	import Button from '$lib/components/ui/Button.svelte';
	import IconButton from '$lib/components/ui/IconButton.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import StatusBadge from '$lib/components/app/StatusBadge.svelte';
	import { selectMenuVariants, selectMenuItemVariants } from '$lib/components/ui/Select.svelte';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ServerStack, ChevronDown, Check, Plus } from '@steeze-ui/heroicons';

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
		value = $bindable(''),
		loading = false,
		error = '',
		canConnect = false,
		onValueChange,
		onConnect,
		onRetry
	}: Props = $props();

	const headingId = $props.id();

	let selected = $derived(servers.find((s) => s.id === value) ?? null);
	let items = $derived(servers.map((s) => ({ value: s.id, label: s.name })));
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
		<div class="flex flex-col items-start gap-2.5 px-3 py-3">
			<p class="text-sm text-muted-foreground">
				{#if canConnect}
					Uploy deploys to machines you own. Connect one to deploy images here.
				{:else}
					Ask a workspace owner to connect a server before deploying images.
				{/if}
			</p>
			{#if canConnect}
				<Button type="button" size="xs" variant="secondary" onclick={onConnect}>
					<Icon src={Plus} theme="outline" class="h-3 w-3" />
					Connect a server
				</Button>
			{/if}
		</div>
	{:else}
		<Select.Root type="single" bind:value {onValueChange} {items}>
			<Select.Trigger
				class="grid w-full cursor-pointer grid-cols-[auto_1fr_auto_auto] items-center gap-x-3 px-3 py-2 text-left text-sm text-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none focus-visible:ring-inset"
				aria-label="Deployment target"
			>
				<span class="text-muted-foreground">
					<Icon src={ServerStack} theme="outline" class="h-4 w-4" />
				</span>
				<span class="truncate">{selected?.name ?? 'Select a server'}</span>
				{#if selected}
					<StatusBadge status={selected.proxy_status} />
				{:else}
					<span></span>
				{/if}
				<Icon src={ChevronDown} theme="outline" class="h-3.5 w-3.5 text-muted-foreground" />
			</Select.Trigger>

			<Select.Portal>
				<Select.Content
					class="{selectMenuVariants()} max-h-60 w-[var(--bits-select-anchor-width)] overflow-auto"
					sideOffset={4}
				>
					<Select.Viewport>
						{#each servers as server (server.id)}
							<Select.Item
								class="{selectMenuItemVariants()} grid grid-cols-[1fr_auto_auto] gap-x-3"
								value={server.id}
								label={server.name}
							>
								{#snippet children({ selected: isSelected })}
									<span class="min-w-0">
										<span class="block truncate">{server.name}</span>
										<span class="block truncate font-mono text-[11px] text-muted-foreground">
											{server.host}:{server.port}
										</span>
									</span>
									<StatusBadge status={server.proxy_status} />
									<span class="grid h-4 w-4 place-content-center" aria-hidden="true">
										{#if isSelected}
											<Icon src={Check} theme="outline" class="h-3.5 w-3.5" />
										{/if}
									</span>
								{/snippet}
							</Select.Item>
						{/each}
					</Select.Viewport>
				</Select.Content>
			</Select.Portal>
		</Select.Root>
	{/if}
</section>

<style>
	.panel {
		box-shadow: var(--shadow-panel);
	}
</style>
