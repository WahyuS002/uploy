<script lang="ts">
	import { page } from '$app/stores';
	import { api } from '$lib/api/client';
	import DeploymentLogs from '$lib/components/DeploymentLogs.svelte';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';

	type ApplicationResponse = components['schemas']['ApplicationResponse'];
	type ApplicationEnvResponse = components['schemas']['ApplicationEnvResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');
	let isOwner = $derived(data.workspace?.role === 'owner');

	let app = $state<ApplicationResponse | null>(null);
	let envs = $state<ApplicationEnvResponse[]>([]);
	let envsLoaded = $state(false);
	let deploymentId = $state<string | null>(null);
	let deploying = $state(false);
	let deployError = $state('');

	// Env form
	let envKey = $state('');
	let envValue = $state('');
	let envError = $state('');

	const appId = $page.params.id as string;

	async function loadApp() {
		const { data } = await api.GET('/api/applications/{id}', {
			params: { path: { id: appId } }
		});
		if (data) app = data;
	}

	async function loadEnvs() {
		const { data, error } = await api.GET('/api/applications/{id}/envs', {
			params: { path: { id: appId } }
		});
		if (data) {
			envs = data;
			envsLoaded = true;
		} else if (error) {
			// 403 for viewers — env section will not render
			envsLoaded = false;
		}
	}

	async function deploy() {
		deployError = '';
		deploying = true;
		try {
			const { data, error } = await api.POST('/api/deployments', {
				body: { application_id: appId }
			});
			if (error) {
				deployError = (error as { error: string }).error;
				return;
			}
			if (data) {
				deploymentId = data.deployment_id;
			}
		} catch {
			deployError = 'Network error';
		} finally {
			deploying = false;
		}
	}

	async function addEnv() {
		envError = '';
		const { data, error } = await api.POST('/api/applications/{id}/envs', {
			params: { path: { id: appId } },
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
		await api.DELETE('/api/applications/{id}/envs/{key}', {
			params: { path: { id: appId, key } }
		});
		envs = envs.filter((e) => e.key !== key);
	}

	$effect(() => {
		loadApp();
		loadEnvs();
	});
</script>

{#if app}
	<section>
		<h2 class="mb-4 text-xl font-bold">{app.name}</h2>

		<div class="mb-4 text-sm text-gray-600">
			<p>Image: {app.image}</p>
			<p>Container: {app.container_name}</p>
			<p>Port: {app.port}</p>
		</div>

		{#if canEdit}
			<!-- Deploy button -->
			<div class="mb-6">
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
			<DeploymentLogs {deploymentId} />
		{/if}

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
									×
								</button>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		{/if}
	</section>
{/if}
