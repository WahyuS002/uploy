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
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogFooter,
		DialogTitle
	} from '$lib/components/ui/dialog';
	import { Server } from '@steeze-ui/heroicons';
	import { cn } from '$lib/components/ui/cn.js';
	import { formatDateTime } from '$lib/format-date';

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
	let latestDeployment = $derived(deployments[0] ?? null);

	// The service only carries a server_id and there is no GET /api/servers/{id},
	// so the list is the only way to put a name on it. Fetched once per mount, not
	// per service: it does not change when the selection does.
	let servers = $state<ServerResponse[]>([]);
	let server = $derived(servers.find((s) => s.id === service.server_id) ?? null);

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

<div class={cn('flex h-full min-h-0 flex-col', className)}>
	<!-- Underline tabs, not filled pills. Four pills across a 420px panel put four
	     competing blocks above content that has none, and the row read as heavier
	     than the thing it was labelling. An underline marks one tab without
	     drawing a box around any of them. -->
	<div class="flex-none border-b border-border bg-card">
		<nav class="flex items-center gap-4 px-5" aria-label="Service sections">
			{#each visibleTabs as tab (tab.id)}
				<button
					type="button"
					onclick={() => (activeTab = tab.id)}
					class={cn(
						// -mb-px pulls the 2px marker onto the nav's own hairline; without it
						// the underline floats a pixel above the border and reads as a
						// misalignment rather than a join.
						'-mb-px cursor-pointer border-b-2 py-2.5 text-xs font-medium whitespace-nowrap transition-colors',
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
			<!-- Current state and the one action that changes it share a row: the status
			     is the reason you'd press Deploy, so putting them apart made you look in
			     two places to decide one thing. -->
			<div class="flex items-center justify-between gap-3">
				<div class="flex min-w-0 items-center gap-2 text-sm">
					{#if latestDeployment}
						<StatusBadge status={latestDeployment.status} class="font-bold" />
						<span class="truncate text-muted-foreground">
							{formatDateTime(latestDeployment.created_at)}
						</span>
					{:else}
						<span class="text-muted-foreground">Never deployed</span>
					{/if}
				</div>
				{#if canEdit}
					<Button onclick={deploy} loading={deploying} size="sm">
						{deploying ? 'Deploying...' : 'Deploy'}
					</Button>
				{/if}
			</div>

			{#if needsRedeploy}
				<Alert tone="warning" class="mt-3">
					Domain configuration changed. Deploy to apply the new routing.
				</Alert>
			{/if}
			{#if deployError}
				<p class="mt-2 text-sm text-destructive">{deployError}</p>
			{/if}
			{#if deploymentId}
				<div class="mt-4">
					<DeploymentLogs {deploymentId} onDone={() => loadDeployments(svcId)} />
				</div>
			{/if}

			<div class="mt-5 border-t border-border pt-4">
				{#if deployments.length === 0}
					<EmptyState
						icon={Server}
						title="No deployments yet"
						description="Trigger your first deployment to see its status and history here."
					/>
				{:else}
					<p class="mb-2 text-xs font-medium text-muted-foreground">History</p>
					<div class="flex flex-col gap-1">
						{#each deployments as dep (dep.id)}
							<div
								class="flex items-center gap-3 rounded-lg border border-border bg-card p-2 text-sm"
							>
								<span class="font-mono text-xs text-muted-foreground">{dep.id.slice(0, 12)}</span>
								<StatusBadge status={dep.status} class="font-bold" />
								<span class="text-muted-foreground">
									{formatDateTime(dep.created_at)}
								</span>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		{:else if activeTab === 'domains'}
			{#if domains.length === 0}
				<p class="mb-3 text-sm text-muted-foreground">No domains attached</p>
			{:else}
				<div class="mb-4 flex flex-col gap-1">
					{#each domains as domain (domain.id)}
						<div
							class="flex items-center gap-3 rounded-lg border border-border bg-card p-2 text-sm"
						>
							<a
								href="https://{domain.domain}"
								target="_blank"
								class="font-medium text-accent underline"
							>
								{domain.domain}
							</a>
							{#if domain.is_primary}
								<Badge tone="info">primary</Badge>
							{/if}
							<StatusBadge status={domain.status} />
							{#if domain.last_error}
								<span class="text-xs text-destructive" title={domain.last_error}>
									{domain.last_error.length > 40
										? domain.last_error.slice(0, 40) + '...'
										: domain.last_error}
								</span>
							{/if}
							{#if canEdit}
								<button
									onclick={() => deleteDomain(domain.id)}
									class="ml-auto cursor-pointer text-destructive hover:text-destructive/80"
									aria-label="Remove domain"
								>
									&times;
								</button>
							{/if}
						</div>
					{/each}
				</div>
			{/if}

			{#if canEdit}
				<form
					onsubmit={(e) => {
						e.preventDefault();
						addDomain();
					}}
					class="flex items-end gap-2"
				>
					<FormField label="Add domain">
						<Input type="text" bind:value={domainInput} placeholder="myapp.example.com" required />
					</FormField>
					<Button type="submit" size="sm" loading={domainAdding}>
						{domainAdding ? 'Adding...' : 'Add'}
					</Button>
				</form>
				{#if domainError}
					<p class="mt-1 text-sm text-destructive">{domainError}</p>
				{/if}
				<Alert tone="neutral" class="mt-2 text-xs">
					<p class="font-medium text-foreground">DNS setup required before deploying:</p>
					<ul class="mt-1 list-inside list-disc space-y-0.5">
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
					class="mb-4 flex gap-2"
				>
					<Input type="text" bind:value={envKey} placeholder="KEY" required class="font-mono" />
					<Input
						type="text"
						bind:value={envValue}
						placeholder="value"
						required
						class="flex-1 font-mono"
					/>
					<Button type="submit" size="sm">Set</Button>
				</form>

				{#if envError}
					<p class="mb-2 text-sm text-destructive">{envError}</p>
				{/if}

				{#if envs.length === 0}
					<p class="text-sm text-muted-foreground">No environment variables set.</p>
				{:else}
					<div class="flex flex-col gap-1">
						{#each envs as env (env.key)}
							<div
								class="flex items-center gap-2 rounded-lg border border-border bg-card p-2 font-mono text-sm"
							>
								<span class="font-bold text-foreground">{env.key}</span>
								<span class="text-muted-foreground">=</span>
								<span class="flex-1 break-all text-muted-foreground">{env.value}</span>
								<button
									onclick={() => deleteEnv(env.key)}
									class="cursor-pointer text-destructive hover:text-destructive/80"
									aria-label="Remove variable"
								>
									&times;
								</button>
							</div>
						{/each}
					</div>
				{/if}
			{:else}
				<p class="text-sm text-muted-foreground">
					Environment variables are only visible to workspace owners and developers.
				</p>
			{/if}
		{:else if activeTab === 'settings'}
			<!-- Sentence-case labels in a fixed left column, not four uppercase headings
			     over four short values. Uppercase + medium weight gave the labels more
			     visual weight than the data they name, which is backwards. `Kind` is
			     gone entirely: the API only accepts "application", so the row could
			     never say anything else. -->
			<dl class="flex flex-col gap-2.5 text-sm">
				<div class="flex items-baseline gap-4">
					<dt class="w-20 flex-none text-muted-foreground">Image</dt>
					<dd class="min-w-0 flex-1 truncate font-mono text-foreground">{service.image}</dd>
				</div>
				<div class="flex items-baseline gap-4">
					<dt class="w-20 flex-none text-muted-foreground">Container</dt>
					<dd class="min-w-0 flex-1 truncate font-mono text-foreground">
						{service.container_name}
					</dd>
				</div>
				<div class="flex items-baseline gap-4">
					<dt class="w-20 flex-none text-muted-foreground">Port</dt>
					<dd class="min-w-0 flex-1 font-mono text-foreground">{service.port}</dd>
				</div>
				<div class="flex items-baseline gap-4">
					<dt class="w-20 flex-none text-muted-foreground">Server</dt>
					<dd class="min-w-0 flex-1 truncate text-foreground">
						{server ? `${server.name} (${server.host})` : '—'}
					</dd>
				</div>
			</dl>

			{#if isOwner}
				<div class="mt-5 border-t border-border pt-4">
					<div class="flex items-start justify-between gap-3">
						<div class="min-w-0">
							<p class="text-sm font-medium text-foreground">Delete service</p>
							<p class="mt-0.5 text-xs text-muted-foreground">
								Removes this service from Uploy. The container already running on the server is left
								alone.
							</p>
						</div>
						<Button variant="destructive" size="sm" onclick={() => (deleteOpen = true)}>
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
