<script lang="ts">
	import { page } from '$app/stores';
	import { api } from '$lib/api/client';
	import DeploymentLogs from '$lib/components/DeploymentLogs.svelte';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import FormField from '$lib/components/app/FormField.svelte';
	import StatusBadge from '$lib/components/app/StatusBadge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import { Server } from '@steeze-ui/heroicons';

	type ServiceResponse = components['schemas']['ServiceResponse'];
	type ServiceDomainResponse = components['schemas']['ServiceDomainResponse'];
	type ServiceEnvResponse = components['schemas']['ServiceEnvResponse'];
	type DeploymentResponse = components['schemas']['DeploymentResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');

	let svc = $derived<ServiceResponse | null>(data.service ?? null);
	let domains = $state<ServiceDomainResponse[]>([]);
	let envs = $state<ServiceEnvResponse[]>([]);
	let envsLoaded = $state(false);
	let deploymentId = $state<string | null>(null);
	let deploying = $state(false);
	let deployError = $state('');
	let deployments = $state<DeploymentResponse[]>([]);

	// Domain form
	let domainInput = $state('');
	let domainError = $state('');
	let domainAdding = $state(false);
	let needsRedeploy = $state(false);

	// Env form
	let envKey = $state('');
	let envValue = $state('');
	let envError = $state('');

	const svcId = $page.params.id as string;

	async function loadDomains() {
		const { data } = await api.GET('/api/services/{id}/domains', {
			params: { path: { id: svcId } }
		});
		if (data) domains = data;
	}

	async function loadEnvs() {
		const { data, error } = await api.GET('/api/services/{id}/envs', {
			params: { path: { id: svcId } }
		});
		if (data) {
			envs = data;
			envsLoaded = true;
		} else if (error) {
			envsLoaded = false;
		}
	}

	async function loadDeployments() {
		const { data } = await api.GET('/api/services/{id}/deployments', {
			params: { path: { id: svcId }, query: { limit: 10 } }
		});
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
				loadDeployments();
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
		loadDomains();
		loadEnvs();
		loadDeployments();
	});
</script>

{#if svc}
	<section>
		<div class="mb-4 text-sm text-muted-foreground">
			<p>Image: {svc.image}</p>
			<p>Container: {svc.container_name}</p>
			<p>Port: {svc.port}</p>
			<p>Kind: {svc.kind}</p>
		</div>

		<!-- Domains Section -->
		<div class="mb-6">
			<h3 class="mb-2 text-lg font-bold text-foreground">Domains</h3>

			{#if domains.length === 0}
				<p class="mb-2 text-sm text-muted-foreground">No domains attached</p>
			{:else}
				<div class="mb-3 flex flex-col gap-1">
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
							For a root domain (e.g. <code>example.com</code>): create an <strong>A record</strong>
							with name <code>@</code> pointing to your server IP
						</li>
					</ul>
				</Alert>
			{/if}
		</div>

		{#if canEdit}
			<!-- Deploy button -->
			<div class="mb-6">
				{#if needsRedeploy}
					<Alert tone="warning" class="mb-2">
						Domain configuration changed. Deploy to apply the new routing.
					</Alert>
				{/if}
				{#if deployError}
					<p class="mb-2 text-sm text-destructive">{deployError}</p>
				{/if}
				<Button onclick={deploy} loading={deploying}>
					{deploying ? 'Deploying...' : 'Deploy'}
				</Button>
			</div>
		{/if}

		<!-- Deployment logs -->
		{#if deploymentId}
			<DeploymentLogs {deploymentId} onDone={() => loadDeployments()} />
		{/if}

		<!-- Deployment History -->
		<div class="mt-6">
			<h3 class="mb-2 text-lg font-bold text-foreground">Deployment History</h3>

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
		</div>

		<!-- Environment Variables (only for owner/developer) -->
		{#if canEdit && envsLoaded}
			<div class="mt-6">
				<h3 class="mb-2 text-lg font-bold text-foreground">Environment Variables</h3>

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
								<span class="flex-1 text-muted-foreground">{env.value}</span>
								<button
									onclick={() => deleteEnv(env.key)}
									class="cursor-pointer text-destructive hover:text-destructive/80"
								>
									&times;
								</button>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		{/if}
	</section>
{/if}
