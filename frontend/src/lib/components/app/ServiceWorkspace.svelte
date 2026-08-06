<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import DeploymentLogs from '$lib/components/DeploymentLogs.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import StatusBadge from '$lib/components/app/StatusBadge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import DataRow from '$lib/components/ui/DataRow.svelte';
	import IconButton from '$lib/components/ui/IconButton.svelte';
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogFooter,
		DialogTitle
	} from '$lib/components/ui/dialog';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { GlobeAlt, Server, XMark } from '@steeze-ui/heroicons';
	import { cn } from '$lib/components/ui/cn.js';
	import { formatDateTime, formatRelativeTime } from '$lib/format-date';

	type ServiceResponse = components['schemas']['ServiceResponse'];
	type ServerResponse = components['schemas']['ServerResponse'];
	type ServiceDomainResponse = components['schemas']['ServiceDomainResponse'];
	type ServiceEnvResponse = components['schemas']['ServiceEnvResponse'];
	type DeploymentResponse = components['schemas']['DeploymentResponse'];

	type Tab = 'deployments' | 'domains' | 'env' | 'settings';

	type Props = {
		service: ServiceResponse;
		canEdit: boolean;
		/** Deleting a service is owner-only server-side, so the action only shows for owners. */
		isOwner?: boolean;
		showEnvVars?: boolean;
		/**
		 * A deployment started outside this panel — the canvas "pending changes"
		 * bar deploys straight from the page and this is how its logs find their
		 * way here. Without it a bar deploy runs invisibly.
		 */
		externalDeploymentId?: string | null;
		onDeleted?: (id: string) => void;
		class?: string;
	};

	let {
		service,
		canEdit,
		isOwner = false,
		showEnvVars = true,
		externalDeploymentId = null,
		onDeleted,
		class: className
	}: Props = $props();

	let svcId = $derived(service.id);

	// Deployments lands first, so it is what a freshly selected service opens on.
	let activeTab = $state<Tab>('deployments');

	let domains = $state<ServiceDomainResponse[]>([]);
	let envs = $state<ServiceEnvResponse[]>([]);
	let envsLoaded = $state(false);
	// Derived rather than synced with an effect: the reset below and an incoming
	// external id would otherwise race on service change, and which one won would
	// come down to declaration order. A deploy started from this panel is the more
	// specific fact, so it wins; otherwise the page's id shows through.
	let localDeploymentId = $state<string | null>(null);
	let deploymentId = $derived(localDeploymentId ?? externalDeploymentId);
	let deploying = $state(false);
	let deployError = $state('');
	let deployments = $state<DeploymentResponse[]>([]);
	// The API returns them newest first, so the head is the current one and the
	// tail is history — the same split the panel shows.
	let latestDeployment = $derived(deployments[0] ?? null);
	let previousDeployments = $derived(deployments.slice(1));

	// The service only carries a server_id and there is no GET /api/servers/{id},
	// so the list is the only way to put a name on it. Fetched once per mount, not
	// per service: it does not change when the selection does.
	let servers = $state<ServerResponse[]>([]);
	let server = $derived(servers.find((s) => s.id === service.server_id) ?? null);

	// The address the Deployments tab leads with. Primary if one is marked, else
	// the first attached — either way it is the one a reader would try.
	let primaryDomain = $derived(domains.find((d) => d.is_primary) ?? domains[0] ?? null);

	let metadata = $derived([
		{ label: 'Image', value: service.image, mono: true },
		{ label: 'Container', value: service.container_name, mono: true },
		{ label: 'Port', value: String(service.port), mono: true },
		{ label: 'Server', value: server ? `${server.name} (${server.host})` : '—', mono: false }
	]);

	let deleteOpen = $state(false);
	let deleting = $state(false);
	let deleteError = $state('');

	let domainInput = $state('');
	let domainError = $state('');
	let domainAdding = $state(false);
	let needsRedeploy = $state(false);

	let envKey = $state('');
	let envValue = $state('');
	let envError = $state('');

	let loadToken = 0;

	async function loadDomains(id: string, token: number) {
		const { data } = await api.GET('/api/services/{id}/domains', {
			params: { path: { id } }
		});
		if (token !== loadToken) return;
		if (data) domains = data;
	}

	async function loadEnvs(id: string, token: number) {
		const { data, error } = await api.GET('/api/services/{id}/envs', {
			params: { path: { id } }
		});
		if (token !== loadToken) return;
		if (data) {
			envs = data;
			envsLoaded = true;
		} else if (error) {
			envsLoaded = false;
		}
	}

	async function loadDeployments(id: string, token: number = loadToken) {
		const { data } = await api.GET('/api/services/{id}/deployments', {
			params: { path: { id }, query: { limit: 10 } }
		});
		if (token !== loadToken) return;
		if (data) deployments = data;
	}

	async function deploy() {
		deployError = '';
		deploying = true;
		needsRedeploy = false;
		try {
			const { data, error } = await api.POST('/api/deployments', {
				body: { service_id: svcId }
			});
			if (error) {
				deployError = (error as { error: string }).error;
				return;
			}
			if (data) {
				localDeploymentId = data.deployment_id;
				loadDeployments(svcId);
			}
		} catch {
			deployError = 'Network error';
		} finally {
			deploying = false;
		}
	}

	async function addDomain() {
		domainError = '';
		domainAdding = true;
		try {
			const { data, error } = await api.POST('/api/services/{id}/domains', {
				params: { path: { id: svcId } },
				body: { domain: domainInput.trim() }
			});
			if (error) {
				domainError = (error as { error: string }).error;
				return;
			}
			if (data) {
				domains = [...domains, data];
				domainInput = '';
				needsRedeploy = true;
			}
		} catch {
			domainError = 'Network error';
		} finally {
			domainAdding = false;
		}
	}

	async function deleteDomain(domainId: string) {
		await api.DELETE('/api/services/{id}/domains/{domainId}', {
			params: { path: { id: svcId, domainId } }
		});
		domains = domains.filter((d) => d.id !== domainId);
		needsRedeploy = true;
	}

	async function addEnv() {
		envError = '';
		const { data, error } = await api.POST('/api/services/{id}/envs', {
			params: { path: { id: svcId } },
			body: { key: envKey, value: envValue }
		});
		if (error) {
			envError = (error as { error: string }).error;
			return;
		}
		if (data) {
			const idx = envs.findIndex((e) => e.key === data.key);
			if (idx >= 0) {
				envs[idx] = data;
				envs = [...envs];
			} else {
				envs = [...envs, data].sort((a, b) => a.key.localeCompare(b.key));
			}
			envKey = '';
			envValue = '';
		}
	}

	async function deleteEnv(key: string) {
		await api.DELETE('/api/services/{id}/envs/{key}', {
			params: { path: { id: svcId, key } }
		});
		envs = envs.filter((e) => e.key !== key);
	}

	async function deleteService() {
		deleteError = '';
		deleting = true;
		try {
			const id = svcId;
			const { error } = await api.DELETE('/api/services/{id}', { params: { path: { id } } });
			if (error) {
				deleteError = (error as { error: string }).error;
				return;
			}
			deleteOpen = false;
			onDeleted?.(id);
		} catch {
			deleteError = 'Network error';
		} finally {
			deleting = false;
		}
	}

	$effect(() => {
		api.GET('/api/servers').then(({ data }) => {
			if (data) servers = data;
		});
	});

	$effect(() => {
		const id = svcId;
		const token = ++loadToken;

		domains = [];
		envs = [];
		envsLoaded = false;
		deployments = [];
		localDeploymentId = null;
		deployError = '';
		deploying = false;
		needsRedeploy = false;

		activeTab = 'deployments';
		domainInput = '';
		domainError = '';
		domainAdding = false;
		envKey = '';
		envValue = '';
		envError = '';
		deleteOpen = false;
		deleting = false;
		deleteError = '';

		loadDomains(id, token);
		loadEnvs(id, token);
		loadDeployments(id, token);
	});

	const tabs: { id: Tab; label: string }[] = [
		{ id: 'deployments', label: 'Deployments' },
		{ id: 'domains', label: 'Domains' },
		{ id: 'env', label: 'Variables' },
		{ id: 'settings', label: 'Settings' }
	];

	let visibleTabs = $derived(tabs.filter((t) => t.id !== 'env' || (showEnvVars && canEdit)));
</script>

<!-- @container, not viewport breakpoints: this same component renders in the
     builder's inspector at clamp(420px, 55vw, 960px) and again on /services/[id]
     at a different width entirely. What the rows should do depends on the room
     *they* have, which the viewport does not know. -->
<div class={cn('@container flex h-full min-h-0 flex-col', className)}>
	<!-- Underline tabs, not filled pills. Four pills across a 420px panel put four
	     competing blocks above content that has none, and the row read as heavier
	     than the thing it was labelling. An underline marks one tab without
	     drawing a box around any of them. -->
	<div class="flex-none border-b border-border bg-card">
		<nav class="flex items-center gap-5 px-5" aria-label="Service sections">
			{#each visibleTabs as tab (tab.id)}
				<button
					type="button"
					onclick={() => (activeTab = tab.id)}
					class={cn(
						// -mb-px pulls the 2px marker onto the nav's own hairline; without it
						// the underline floats a pixel above the border and reads as a
						// misalignment rather than a join.
						'-mb-px cursor-pointer border-b-2 py-3.5 text-[15px] font-medium whitespace-nowrap transition-colors',
						'focus-visible:rounded-sm focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-card focus-visible:outline-none',
						activeTab === tab.id
							? 'border-foreground text-foreground'
							: 'border-transparent text-muted-foreground hover:text-foreground'
					)}
					aria-current={activeTab === tab.id ? 'page' : undefined}
				>
					{tab.label}
				</button>
			{/each}
		</nav>
	</div>

	<div class="min-h-0 flex-1 overflow-y-auto bg-card px-5 py-5">
		{#if activeTab === 'deployments'}
			<!-- Where the thing is reachable, and the one action that changes what's
			     running, on one line. The address is the reason you'd press Deploy, so
			     splitting them made you look in two places to decide one thing. -->
			<div class="flex items-center justify-between gap-3">
				<div class="flex min-w-0 items-center gap-1.5 text-[13px] text-muted-foreground">
					<Icon src={GlobeAlt} theme="outline" class="h-3.5 w-3.5 flex-none" />
					{#if primaryDomain}
						<a
							href="https://{primaryDomain.domain}"
							target="_blank"
							rel="noreferrer"
							class="truncate text-foreground hover:underline"
						>
							{primaryDomain.domain}
						</a>
						{#if domains.length > 1}
							<span class="flex-none">+{domains.length - 1}</span>
						{/if}
					{:else}
						<span class="truncate">No domain attached</span>
					{/if}
				</div>
				{#if canEdit}
					<Button onclick={deploy} loading={deploying} size="sm">
						{deploying ? 'Deploying...' : 'Deploy'}
					</Button>
				{/if}
			</div>

			{#if needsRedeploy}
				<Alert tone="warning" class="mt-3 text-[13px]">
					Domain configuration changed. Deploy to apply the new routing.
				</Alert>
			{/if}
			{#if deployError}
				<p class="mt-2 text-[15px] text-destructive">{deployError}</p>
			{/if}

			{#if latestDeployment || deploymentId}
				<!-- The current deployment and its output are one object, so they share
				     one card: the log panel drops its own edge and joins on below. Two
				     stacked outlines read as two unrelated things stacked by accident. -->
				<div class="mt-4 overflow-hidden rounded-lg border border-border">
					<!-- Narrow, the meta stacks under the image because there is no room for it
					     anywhere else. Once the panel is wide the same two facts sit on one
					     line, image left and meta right — which is what stops a 960px card
					     from being two short lines against half a metre of nothing. -->
					<div class="flex items-start gap-2.5 p-3 @2xl:items-center">
						<StatusBadge
							status={latestDeployment?.status ?? 'in_progress'}
							class="mt-px flex-none @2xl:mt-0"
						/>
						<div
							class="min-w-0 flex-1 @2xl:flex @2xl:items-baseline @2xl:justify-between @2xl:gap-6"
						>
							<p class="truncate font-mono text-[15px] text-foreground">{service.image}</p>
							<p
								class="mt-0.5 truncate text-[13px] text-muted-foreground @2xl:mt-0 @2xl:flex-none"
							>
								{#if latestDeployment}
									<!-- Relative first: "2 hours ago" is what you actually want to know
									     about a deployment. The exact timestamp stays one hover away. -->
									<span title={formatDateTime(latestDeployment.created_at)}>
										{formatRelativeTime(latestDeployment.created_at)}
									</span>
									<span class="px-1">·</span>
									<span class="font-mono">{latestDeployment.id.slice(0, 8)}</span>
								{:else}
									Starting...
								{/if}
							</p>
						</div>
					</div>
					{#if deploymentId}
						<DeploymentLogs {deploymentId} flush onDone={() => loadDeployments(svcId)} />
					{/if}
				</div>
			{:else}
				<div class="mt-4 rounded-lg border border-dashed border-border">
					<EmptyState
						icon={Server}
						title="No deployments yet"
						description="Deploy this service to see its status, live logs and history here."
						class="px-5 py-8"
					/>
				</div>
			{/if}

			{#if previousDeployments.length > 0}
				<!-- mt-8, not mt-5: history is a different subject from what is running
				     now, and at 20px it sat closer to the current card than the card's own
				     rows sit to each other — proximity said "same thing" while the label
				     said otherwise. -->
				<div class="mt-8">
					<p class="mb-2 text-[13px] text-muted-foreground">Previous</p>
					<div class="overflow-hidden rounded-lg border border-border">
						{#each previousDeployments as dep (dep.id)}
							<DataRow density="dense" class="gap-2.5 text-[13px]">
								<StatusBadge status={dep.status} class="flex-none" />
								<span class="truncate font-mono text-muted-foreground">{dep.id.slice(0, 8)}</span>
								<span
									class="ml-auto flex-none text-muted-foreground"
									title={formatDateTime(dep.created_at)}
								>
									{formatRelativeTime(dep.created_at)}
								</span>
							</DataRow>
						{/each}
					</div>
				</div>
			{/if}
		{:else if activeTab === 'domains'}
			{#if domains.length === 0}
				<div class="rounded-lg border border-dashed border-border px-4 py-5 text-center">
					<p class="text-[15px] text-muted-foreground">No domains attached</p>
					<p class="mt-1 text-[13px] text-muted-foreground">
						Add one below to reach this service by name instead of by port.
					</p>
				</div>
			{:else}
				<!-- One divided card, not a stack of separate bordered pills: these are
				     rows of the same list, and giving each its own edge made a three-domain
				     panel read as three unrelated objects. -->
				<div class="overflow-hidden rounded-lg border border-border">
					{#each domains as domain (domain.id)}
						<DataRow density="dense" class="items-start gap-2.5">
							<div class="min-w-0 flex-1">
								<div class="flex min-w-0 items-center gap-2">
									<a
										href="https://{domain.domain}"
										target="_blank"
										rel="noreferrer"
										class="truncate text-[15px] font-medium text-foreground hover:underline"
									>
										{domain.domain}
									</a>
									{#if domain.is_primary}
										<Badge tone="info" class="flex-none">primary</Badge>
									{/if}
								</div>
								{#if domain.last_error}
									<p class="mt-1 truncate text-[13px] text-destructive" title={domain.last_error}>
										{domain.last_error}
									</p>
								{/if}
							</div>
							<StatusBadge status={domain.status} class="mt-px flex-none" />
							{#if canEdit}
								<IconButton
									variant="ghost"
									onclick={() => deleteDomain(domain.id)}
									class="-mr-1 flex-none hover:text-destructive"
									aria-label="Remove {domain.domain}"
								>
									<Icon src={XMark} theme="outline" class="h-3.5 w-3.5" />
								</IconButton>
							{/if}
						</DataRow>
					{/each}
				</div>
			{/if}

			{#if canEdit}
				<form
					onsubmit={(e) => {
						e.preventDefault();
						addDomain();
					}}
					class="mt-4 flex items-end gap-2"
				>
					<FormField label="Add domain">
						<Input type="text" bind:value={domainInput} placeholder="myapp.example.com" required />
					</FormField>
					<Button type="submit" size="sm" loading={domainAdding}>
						{domainAdding ? 'Adding...' : 'Add'}
					</Button>
				</form>
				{#if domainError}
					<p class="mt-1 text-[15px] text-destructive">{domainError}</p>
				{/if}
				<Alert tone="neutral" class="mt-3 text-[13px]">
					<p class="font-medium text-foreground">DNS setup required before deploying:</p>
					<!-- Prose, so it takes a measure. Uncapped it ran ~105ch in a 960px
					     panel, well past the 65–75ch a reader tracks without losing the line. -->
					<ul class="mt-1 max-w-[68ch] list-inside list-disc space-y-0.5">
						<li>
							For a subdomain (e.g. <code>app.example.com</code>): create an
							<strong>A record</strong>
							with name <code>app</code> pointing to your server IP
						</li>
						<li>
							For a root domain (e.g. <code>example.com</code>): create an
							<strong>A record</strong>
							with name <code>@</code> pointing to your server IP
						</li>
					</ul>
				</Alert>
			{/if}
		{:else if activeTab === 'env'}
			{#if canEdit && envsLoaded}
				<form
					onsubmit={(e) => {
						e.preventDefault();
						addEnv();
					}}
					class="mb-4 flex items-center gap-2"
				>
					<!-- A third of the row for the key, the rest for the value: at 420px the
					     two inputs were splitting evenly and a value like a connection string
					     got the same room as `PORT`. -->
					<Input
						type="text"
						bind:value={envKey}
						placeholder="KEY"
						required
						size="sm"
						class="w-28 flex-none font-mono"
					/>
					<Input
						type="text"
						bind:value={envValue}
						placeholder="value"
						required
						size="sm"
						class="min-w-0 flex-1 font-mono"
					/>
					<Button type="submit" size="sm" class="flex-none">Set</Button>
				</form>

				{#if envError}
					<p class="mb-2 text-[15px] text-destructive">{envError}</p>
				{/if}

				{#if envs.length === 0}
					<div class="rounded-lg border border-dashed border-border px-4 py-5 text-center">
						<p class="text-[15px] text-muted-foreground">No variables set</p>
						<p class="mt-1 text-[13px] text-muted-foreground">
							They are passed to the container on the next deployment.
						</p>
					</div>
				{:else}
					<!-- Key over value, not key = value on one line: a 420px panel wrapped
					     every real secret mid-string, and the wrapped tail lined up under
					     the key it did not belong to. Stacked, the key stays scannable and
					     the value gets the full width to break in. -->
					<div class="overflow-hidden rounded-lg border border-border">
						{#each envs as env (env.key)}
							<DataRow density="dense" class="items-start gap-2.5">
								<div class="min-w-0 flex-1 font-mono">
									<p class="truncate text-[15px] font-medium text-foreground">{env.key}</p>
									<p class="mt-0.5 text-[13px] break-all text-muted-foreground">{env.value}</p>
								</div>
								<IconButton
									variant="ghost"
									onclick={() => deleteEnv(env.key)}
									class="-mr-1 flex-none hover:text-destructive"
									aria-label="Remove {env.key}"
								>
									<Icon src={XMark} theme="outline" class="h-3.5 w-3.5" />
								</IconButton>
							</DataRow>
						{/each}
					</div>
				{/if}
			{:else}
				<p class="text-[15px] text-muted-foreground">
					Environment variables are only visible to workspace owners and developers.
				</p>
			{/if}
		{:else if activeTab === 'settings'}
			<!-- Sentence-case labels in a fixed left column, not four uppercase headings
			     over four short values. Uppercase + medium weight gave the labels more
			     visual weight than the data they name, which is backwards. `Kind` is
			     gone entirely: the API only accepts "application", so the row could
			     never say anything else. The card is what turns four floating pairs
			     into one block you can read as "this is the service". -->
			<dl class="overflow-hidden rounded-lg border border-border">
				{#each metadata as row (row.label)}
					<DataRow density="dense" class="gap-4 text-[15px]">
						<dt class="w-20 flex-none text-muted-foreground">{row.label}</dt>
						<dd class={cn('min-w-0 flex-1 truncate text-foreground', row.mono && 'font-mono')}>
							{row.value}
						</dd>
					</DataRow>
				{/each}
			</dl>

			{#if isOwner}
				<div class="mt-5">
					<p class="mb-2 text-[13px] text-muted-foreground">Danger zone</p>
					<!-- Tinted edge, plain surface: the border is enough to say "read this
					     twice", and a filled red block for an action nobody has taken yet
					     reads as an error that already happened. -->
					<div
						class="flex items-start justify-between gap-3 rounded-lg border border-destructive/25 p-3"
					>
						<div class="min-w-0">
							<p class="text-[15px] font-medium text-foreground">Delete service</p>
							<p class="mt-0.5 text-[13px] text-muted-foreground">
								Removes this service from Uploy. The container already running on the server is left
								alone.
							</p>
						</div>
						<Button
							variant="destructive"
							size="sm"
							class="flex-none"
							onclick={() => (deleteOpen = true)}
						>
							Delete
						</Button>
					</div>
				</div>
			{/if}
		{/if}
	</div>
</div>

<Dialog bind:open={deleteOpen}>
	<DialogContent>
		<DialogHeader>
			<DialogTitle>Delete {service.name}?</DialogTitle>
		</DialogHeader>
		<div class="px-5 pb-5 text-sm text-muted-foreground">
			Its domains and environment variables go with it. This cannot be undone.
			{#if deleteError}
				<p class="mt-2 text-destructive">{deleteError}</p>
			{/if}
		</div>
		<DialogFooter>
			<Button type="button" variant="secondary" size="sm" onclick={() => (deleteOpen = false)}>
				Cancel
			</Button>
			<Button
				type="button"
				variant="destructive"
				size="sm"
				loading={deleting}
				onclick={deleteService}
			>
				{deleting ? 'Deleting...' : 'Delete'}
			</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>
