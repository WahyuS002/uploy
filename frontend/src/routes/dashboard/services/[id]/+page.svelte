<script lang="ts">
	import { page } from '$app/stores';
	import { api } from '$lib/api/client';
	import DeploymentLogs from '$lib/components/DeploymentLogs.svelte';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';

	type ServiceResponse = components['schemas']['ServiceResponse'];
	type ServiceDomainResponse = components['schemas']['ServiceDomainResponse'];
	type ServiceEnvResponse = components['schemas']['ServiceEnvResponse'];
	type DeploymentResponse = components['schemas']['DeploymentResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');
	let isOwner = $derived(data.workspace?.role === 'owner');

	let svc = $state<ServiceResponse | null>(null);
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

	async function loadService() {
		const { data } = await api.GET('/api/services/{id}', {
			params: { path: { id: svcId } }
		});
		if (data) svc = data;
	}

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
		loadService();
		loadDomains();
		loadEnvs();
		loadDeployments();
	});
</script>

{#if svc}
	<section>
		<h2 class="mb-4 text-xl font-bold">{svc.name}</h2>

		<div class="mb-4 text-sm text-gray-600">
			<p>Image: {svc.image}</p>
			<p>Container: {svc.container_name}</p>
			<p>Port: {svc.port}</p>
			<p>Kind: {svc.kind}</p>
		</div>

		<!-- Domains Section -->
		<div class="mb-6">
			<h3 class="mb-2 text-lg font-bold">Domains</h3>

			{#if domains.length === 0}
				<p class="mb-2 text-sm text-gray-400">No domains attached</p>
			{:else}
				<div class="mb-3 flex flex-col gap-1">
					{#each domains as domain}
						<div class="flex items-center gap-3 rounded border p-2 text-sm">
							<a href="https://{domain.domain}" target="_blank" class="font-medium text-blue-600 underline">
								{domain.domain}
							</a>
							{#if domain.is_primary}
								<span class="rounded bg-blue-100 px-1.5 py-0.5 text-xs font-medium text-blue-700">primary</span>
							{/if}
							<span
								class="rounded px-2 py-0.5 text-xs font-medium"
								class:bg-green-100={domain.status === 'ready'}
								class:text-green-700={domain.status === 'ready'}
								class:bg-yellow-100={domain.status === 'pending'}
								class:text-yellow-700={domain.status === 'pending'}
								class:bg-red-100={domain.status === 'error'}
								class:text-red-700={domain.status === 'error'}
							>
								{domain.status}
							</span>
							{#if domain.last_error}
								<span class="text-xs text-red-500" title={domain.last_error}>
									{domain.last_error.length > 40
										? domain.last_error.slice(0, 40) + '...'
										: domain.last_error}
								</span>
							{/if}
							{#if canEdit}
								<button
									onclick={() => deleteDomain(domain.id)}
									class="ml-auto cursor-pointer text-red-500 hover:text-red-700"
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
					onsubmit={(e) => { e.preventDefault(); addDomain(); }}
					class="flex items-end gap-2"
				>
					<label class="flex flex-col gap-1 text-sm">
						<span>Add domain</span>
						<input
							type="text"
							bind:value={domainInput}
							placeholder="myapp.example.com"
							required
							class="rounded border p-1 text-sm"
						/>
					</label>
					<button
						type="submit"
						disabled={domainAdding}
						class="cursor-pointer rounded-sm bg-black px-3 py-1.5 text-sm text-white disabled:opacity-50"
					>
						{domainAdding ? 'Adding...' : 'Add'}
					</button>
				</form>
				{#if domainError}
					<p class="mt-1 text-sm text-red-600">{domainError}</p>
				{/if}
				<div class="mt-2 rounded border border-gray-200 bg-gray-50 p-2 text-xs text-gray-500">
					<p class="font-medium text-gray-600">DNS setup required before deploying:</p>
					<ul class="mt-1 list-inside list-disc space-y-0.5">
						<li>For a subdomain (e.g. <code>app.example.com</code>): create an <strong>A record</strong> with name <code>app</code> pointing to your server IP</li>
						<li>For a root domain (e.g. <code>example.com</code>): create an <strong>A record</strong> with name <code>@</code> pointing to your server IP</li>
					</ul>
				</div>
			{/if}
		</div>

		{#if canEdit}
			<!-- Deploy button -->
			<div class="mb-6">
				{#if needsRedeploy}
					<div class="mb-2 rounded border border-yellow-300 bg-yellow-50 p-2 text-sm text-yellow-700">
						Domain configuration changed. Deploy to apply the new routing.
					</div>
				{/if}
				{#if deployError}
					<p class="mb-2 text-sm text-red-600">{deployError}</p>
				{/if}
				<button
					onclick={deploy}
					disabled={deploying}
					class="cursor-pointer rounded-sm bg-black px-4 py-2 text-white disabled:opacity-50"
				>
					{deploying ? 'Deploying...' : 'Deploy'}
				</button>
			</div>
		{/if}

		<!-- Deployment logs -->
		{#if deploymentId}
			<DeploymentLogs {deploymentId} onDone={() => loadDeployments()} />
		{/if}

		<!-- Deployment History -->
		<div class="mt-6">
			<h3 class="mb-2 text-lg font-bold">Deployment History</h3>

			{#if deployments.length === 0}
				<p class="text-sm text-gray-500">No deployments yet.</p>
			{:else}
				<div class="flex flex-col gap-1">
					{#each deployments as dep}
						<div class="flex items-center gap-3 rounded border p-2 text-sm">
							<span class="font-mono text-xs text-gray-400">{dep.id.slice(0, 12)}</span>
							<span
								class="rounded px-2 py-0.5 text-xs font-bold"
								class:bg-green-100={dep.status === 'success'}
								class:text-green-800={dep.status === 'success'}
								class:bg-red-100={dep.status === 'failed'}
								class:text-red-800={dep.status === 'failed'}
								class:bg-yellow-100={dep.status === 'in_progress'}
								class:text-yellow-800={dep.status === 'in_progress'}
							>
								{dep.status}
							</span>
							<span class="text-gray-500">
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
				<h3 class="mb-2 text-lg font-bold">Environment Variables</h3>

				<form
					onsubmit={(e) => { e.preventDefault(); addEnv(); }}
					class="mb-4 flex gap-2"
				>
					<input
						type="text"
						bind:value={envKey}
						placeholder="KEY"
						required
						class="rounded border p-1 font-mono text-sm"
					/>
					<input
						type="text"
						bind:value={envValue}
						placeholder="value"
						required
						class="flex-1 rounded border p-1 font-mono text-sm"
					/>
					<button
						type="submit"
						class="cursor-pointer rounded-sm bg-black px-3 py-1 text-sm text-white"
					>
						Set
					</button>
				</form>

				{#if envError}
					<p class="mb-2 text-sm text-red-600">{envError}</p>
				{/if}

				{#if envs.length === 0}
					<p class="text-sm text-gray-500">No environment variables set.</p>
				{:else}
					<div class="flex flex-col gap-1">
						{#each envs as env}
							<div class="flex items-center gap-2 rounded border p-2 font-mono text-sm">
								<span class="font-bold">{env.key}</span>
								<span class="text-gray-400">=</span>
								<span class="flex-1 text-gray-600">{env.value}</span>
								<button
									onclick={() => deleteEnv(env.key)}
									class="cursor-pointer text-red-500 hover:text-red-700"
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
