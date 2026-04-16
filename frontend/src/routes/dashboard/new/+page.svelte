<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { Server, Check, Squares2x2 } from '@steeze-ui/heroicons';
	import { Container } from 'lucide-svelte';

	type ServerResponse = components['schemas']['ServerResponse'];
	type ProjectResponse = components['schemas']['ProjectResponse'];
	type EnvironmentResponse = components['schemas']['EnvironmentResponse'];

	type Starter = 'empty-project' | 'docker-image';

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');
	let isOwner = $derived(data.workspace?.role === 'owner');

	let servers = $state<ServerResponse[]>([]);
	let loading = $state(true);
	let selectedServerId = $state('');
	let busyStarter = $state<Starter | null>(null);
	let error = $state('');

	let selectedServer = $derived(servers.find((s) => s.id === selectedServerId));

	function autoProjectName(): string {
		const d = new Date();
		const pad = (n: number) => n.toString().padStart(2, '0');
		const date = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
		const time = `${pad(d.getHours())}:${pad(d.getMinutes())}`;
		return `Untitled Project ${date} ${time}`;
	}

	async function load() {
		loading = true;
		try {
			const serversRes = await api.GET('/api/servers');
			if (serversRes.data) {
				servers = serversRes.data;
				if (servers.length === 1) selectedServerId = servers[0].id;
			}
		} finally {
			loading = false;
		}
	}

	async function ensureProject(): Promise<ProjectResponse | null> {
		const { data, error: err } = await api.POST('/api/projects', {
			body: { name: autoProjectName() }
		});
		if (err) {
			error = (err as { error: string }).error ?? 'Failed to create project';
			return null;
		}
		return data ?? null;
	}

	async function ensureEnvironment(projectId: string): Promise<EnvironmentResponse | null> {
		const existing = await api.GET('/api/projects/{id}/environments', {
			params: { path: { id: projectId } }
		});
		if (existing.data && existing.data.length > 0) {
			const prod = existing.data.find((e) => e.name === 'production');
			return prod ?? existing.data[0];
		}
		const { data, error: err } = await api.POST('/api/projects/{id}/environments', {
			params: { path: { id: projectId } },
			body: { name: 'production' }
		});
		if (err) {
			error = (err as { error: string }).error ?? 'Failed to create environment';
			return null;
		}
		return data ?? null;
	}

	async function runEmptyProject() {
		error = '';
		busyStarter = 'empty-project';
		try {
			const project = await ensureProject();
			if (!project) return;
			// eslint-disable-next-line svelte/no-navigation-without-resolve
			await goto(`/dashboard/projects/${project.id}`);
		} catch {
			error = 'Network error';
		} finally {
			busyStarter = null;
		}
	}

	async function runDockerImage() {
		if (!selectedServerId) return;
		error = '';
		busyStarter = 'docker-image';
		try {
			const project = await ensureProject();
			if (!project) return;
			const envRow = await ensureEnvironment(project.id);
			if (!envRow) return;
			const params = new URLSearchParams({
				starter: 'docker-image',
				serverId: selectedServerId,
				projectId: project.id,
				environmentId: envRow.id
			});
			// eslint-disable-next-line svelte/no-navigation-without-resolve
			await goto(`/dashboard/services?${params.toString()}`);
		} catch {
			error = 'Network error';
		} finally {
			busyStarter = null;
		}
	}

	$effect(() => {
		load();
	});
</script>

<section>
	<PageHeader
		title="New"
		description="Pick a server, then choose how you want to start."
	/>

	{#if loading}
		<div class="flex flex-col gap-6">
			<div class="h-32 animate-pulse rounded-xl bg-surface-muted"></div>
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div class="h-40 animate-pulse rounded-xl bg-surface-muted"></div>
				<div class="h-40 animate-pulse rounded-xl bg-surface-muted"></div>
			</div>
		</div>
	{:else if !canEdit}
		<EmptyState
			icon={Server}
			title="You don't have permission to create projects"
			description="Ask a workspace owner or developer to create a project, or request a role change."
		>
			{#snippet actions()}
				<Button href="/dashboard/projects" variant="secondary" size="sm">Back to projects</Button>
			{/snippet}
		</EmptyState>
	{:else if servers.length === 0 && isOwner}
		<EmptyState
			icon={Server}
			title="Connect a server to get started"
			description="Uploy deploys to your own infrastructure. Add your first server, then come back here to start a project."
		>
			{#snippet actions()}
				<Button href="/dashboard/servers?returnTo=/dashboard/new" size="sm">
					Add a server
				</Button>
				<Button href="/dashboard/projects" variant="secondary" size="sm">Cancel</Button>
			{/snippet}
		</EmptyState>
	{:else if servers.length === 0}
		<EmptyState
			icon={Server}
			title="No servers available"
			description="A workspace owner needs to connect a server before you can create a project. Ask an owner to add one."
		>
			{#snippet actions()}
				<Button href="/dashboard/projects" variant="secondary" size="sm">Back to projects</Button>
			{/snippet}
		</EmptyState>
	{:else}
		<!-- Step 1: Server -->
		<div class="mb-8">
			<div class="mb-3 flex items-baseline justify-between">
				<h3 class="text-sm font-semibold text-foreground">
					<span class="mr-2 inline-flex h-5 w-5 items-center justify-center rounded-full bg-foreground text-[11px] font-semibold text-primary-foreground">1</span>
					Choose a server
				</h3>
				{#if isOwner}
					<!-- eslint-disable svelte/no-navigation-without-resolve -->
					<a
						href="/dashboard/servers?returnTo=/dashboard/new"
						class="text-xs text-muted-foreground hover:text-foreground"
					>
						+ Add server
					</a>
					<!-- eslint-enable svelte/no-navigation-without-resolve -->
				{/if}
			</div>
			<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
				{#each servers as server (server.id)}
					{@const selected = server.id === selectedServerId}
					<button
						type="button"
						onclick={() => (selectedServerId = server.id)}
						class="group flex items-start gap-3 rounded-xl border bg-surface p-4 text-left transition-colors {selected
							? 'border-foreground ring-1 ring-foreground'
							: 'border-border hover:border-foreground/40'}"
					>
						<div class="mt-0.5 grid h-9 w-9 place-content-center rounded-lg bg-surface-muted">
							<Icon src={Server} theme="outline" class="h-4 w-4 text-foreground" />
						</div>
						<div class="min-w-0 flex-1">
							<div class="flex items-center justify-between gap-2">
								<span class="truncate font-medium text-foreground">{server.name}</span>
								{#if selected}
									<Icon src={Check} theme="outline" class="h-4 w-4 text-foreground" />
								{/if}
							</div>
							<div class="mt-0.5 truncate font-mono text-xs text-muted-foreground">
								{server.ssh_user}@{server.host}:{server.port}
							</div>
						</div>
					</button>
				{/each}
			</div>
		</div>

		<!-- Step 2: Starter -->
		<div>
			<h3 class="mb-3 text-sm font-semibold text-foreground">
				<span class="mr-2 inline-flex h-5 w-5 items-center justify-center rounded-full bg-foreground text-[11px] font-semibold text-primary-foreground">2</span>
				Pick a starter
			</h3>
			{#if !selectedServerId}
				<p class="mb-3 text-xs text-muted-foreground">Select a server above to unlock starters.</p>
			{/if}
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div
					class="flex flex-col rounded-xl border border-border bg-surface p-5 transition-opacity {selectedServerId
						? ''
						: 'pointer-events-none opacity-50'}"
				>
					<div class="mb-4 grid h-10 w-10 place-content-center rounded-lg bg-surface-muted">
						<Icon src={Squares2x2} theme="outline" class="h-5 w-5 text-foreground" />
					</div>
					<h4 class="font-semibold text-foreground">Empty Project</h4>
					<p class="mt-1 flex-1 text-sm text-muted-foreground">
						Create an empty project to organise services and environments later.
					</p>
					<div class="mt-5">
						<Button
							size="sm"
							onclick={runEmptyProject}
							loading={busyStarter === 'empty-project'}
							disabled={!selectedServerId || busyStarter !== null}
						>
							{busyStarter === 'empty-project' ? 'Creating...' : 'Create project'}
						</Button>
					</div>
				</div>

				<div
					class="flex flex-col rounded-xl border border-border bg-surface p-5 transition-opacity {selectedServerId
						? ''
						: 'pointer-events-none opacity-50'}"
				>
					<div class="mb-4 grid h-10 w-10 place-content-center rounded-lg bg-surface-muted">
						<Container class="h-5 w-5 text-foreground" strokeWidth={1.75} />
					</div>
					<h4 class="font-semibold text-foreground">Docker Image</h4>
					<p class="mt-1 flex-1 text-sm text-muted-foreground">
						Deploy a prebuilt image from a registry onto {selectedServer?.name ?? 'your server'}.
					</p>
					<div class="mt-5">
						<Button
							size="sm"
							onclick={runDockerImage}
							loading={busyStarter === 'docker-image'}
							disabled={!selectedServerId || busyStarter !== null}
						>
							{busyStarter === 'docker-image' ? 'Preparing...' : 'Continue'}
						</Button>
					</div>
				</div>
			</div>

			{#if error}
				<p class="mt-4 text-sm text-danger">{error}</p>
			{/if}
		</div>
	{/if}
</section>
