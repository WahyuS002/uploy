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
	import { Server } from '@steeze-ui/heroicons';
	import { cn } from '$lib/components/ui/cn.js';

	type ServiceResponse = components['schemas']['ServiceResponse'];
	type ServiceDomainResponse = components['schemas']['ServiceDomainResponse'];
	type ServiceEnvResponse = components['schemas']['ServiceEnvResponse'];
	type DeploymentResponse = components['schemas']['DeploymentResponse'];

	type Tab = 'overview' | 'domains' | 'env' | 'deployments';

	type Props = {
		service: ServiceResponse;
		canEdit: boolean;
		showEnvVars?: boolean;
		class?: string;
	};

	let { service, canEdit, showEnvVars = true, class: className }: Props = $props();

	let svcId = $derived(service.id);

	let activeTab = $state<Tab>('overview');

	let domains = $state<ServiceDomainResponse[]>([]);
	let envs = $state<ServiceEnvResponse[]>([]);
	let envsLoaded = $state(false);
	let deploymentId = $state<string | null>(null);
	let deploying = $state(false);
	let deployError = $state('');
	let deployments = $state<DeploymentResponse[]>([]);

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
				deploymentId = data.deployment_id;
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

	$effect(() => {
		const id = svcId;
		const token = ++loadToken;

		domains = [];
		envs = [];
		envsLoaded = false;
		deployments = [];
		deploymentId = null;
		deployError = '';
		deploying = false;
		needsRedeploy = false;

		activeTab = 'overview';
		domainInput = '';
		domainError = '';
		domainAdding = false;
		envKey = '';
		envValue = '';
		envError = '';

		loadDomains(id, token);
		loadEnvs(id, token);
		loadDeployments(id, token);
	});

	const tabs: { id: Tab; label: string }[] = [
		{ id: 'overview', label: 'Overview' },
		{ id: 'domains', label: 'Domains' },
		{ id: 'env', label: 'Env Vars' },
		{ id: 'deployments', label: 'Deployments' }
	];

	let visibleTabs = $derived(tabs.filter((t) => t.id !== 'env' || (showEnvVars && canEdit)));
</script>

<div class={cn('flex h-full min-h-0 flex-col', className)}>
	<div class="flex-none border-b border-border bg-card">
		<nav class="flex items-center gap-1 px-3 pt-2" aria-label="Service sections">
			{#each visibleTabs as tab (tab.id)}
				<button
					type="button"
					onclick={() => (activeTab = tab.id)}
					class={cn(
						'cursor-pointer rounded-md px-3 py-1.5 text-xs font-medium transition-colors',
						activeTab === tab.id
							? 'bg-accent text-accent-foreground'
							: 'text-muted-foreground hover:bg-accent/60 hover:text-accent-foreground'
					)}
					aria-current={activeTab === tab.id ? 'page' : undefined}
				>
					{tab.label}
				</button>
			{/each}
		</nav>
		<div class="h-2"></div>
	</div>

	<div class="min-h-0 flex-1 overflow-y-auto bg-card px-5 py-5">
		{#if activeTab === 'overview'}
			<dl class="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
				<div>
					<dt class="text-xs font-medium text-muted-foreground uppercase">Image</dt>
					<dd class="mt-1 truncate font-mono text-foreground">{service.image}</dd>
				</div>
				<div>
					<dt class="text-xs font-medium text-muted-foreground uppercase">Container</dt>
					<dd class="mt-1 truncate font-mono text-foreground">{service.container_name}</dd>
				</div>
				<div>
					<dt class="text-xs font-medium text-muted-foreground uppercase">Port</dt>
					<dd class="mt-1 font-mono text-foreground">{service.port}</dd>
				</div>
				<div>
					<dt class="text-xs font-medium text-muted-foreground uppercase">Kind</dt>
					<dd class="mt-1 font-mono text-foreground">{service.kind}</dd>
				</div>
			</dl>

			{#if canEdit}
				<div class="mt-6 border-t border-border pt-5">
					<div class="flex items-center justify-between gap-3">
						<div>
							<h4 class="text-sm font-semibold text-foreground">Deploy</h4>
							<p class="text-xs text-muted-foreground">
								Trigger a new deployment for this service.
							</p>
						</div>
						<Button onclick={deploy} loading={deploying} size="sm">
							{deploying ? 'Deploying...' : 'Deploy'}
						</Button>
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
				</div>
			{/if}
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
		{:else if activeTab === 'deployments'}
			{#if deployments.length === 0}
				<EmptyState
					icon={Server}
					title="No deployments yet"
					description="Trigger your first deployment to see its status and history here."
				/>
			{:else}
				<div class="flex flex-col gap-1">
					{#each deployments as dep (dep.id)}
						<div
							class="flex items-center gap-3 rounded-lg border border-border bg-card p-2 text-sm"
						>
							<span class="font-mono text-xs text-muted-foreground">{dep.id.slice(0, 12)}</span>
							<StatusBadge status={dep.status} class="font-bold" />
							<span class="text-muted-foreground">
								{new Date(dep.created_at).toLocaleString()}
							</span>
						</div>
					{/each}
				</div>
			{/if}
		{/if}
	</div>
</div>
