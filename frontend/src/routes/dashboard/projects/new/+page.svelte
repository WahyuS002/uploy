<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import Button from '$lib/components/ui/Button.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import { Icon, type IconSource } from '@steeze-ui/svelte-icon';
	import {
		Server,
		Squares2x2,
		CodeBracket,
		CircleStack,
		RectangleStack
	} from '@steeze-ui/heroicons';
	import { Container } from 'lucide-svelte';

	type ProjectResponse = components['schemas']['ProjectResponse'];

	type Starter = 'empty-project' | 'docker-image';

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');

	let busyStarter = $state<Starter | null>(null);
	let error = $state('');

	async function createProject(): Promise<ProjectResponse | null> {
		const { data, error: err } = await api.POST('/api/projects', {
			body: {}
		});
		if (err) {
			error = (err as { error: string }).error ?? 'Failed to create project';
			return null;
		}
		return data ?? null;
	}

	async function launch(starter: Starter) {
		if (busyStarter) return;
		error = '';
		busyStarter = starter;
		try {
			const project = await createProject();
			if (!project) return;
			const target =
				starter === 'docker-image'
					? `/dashboard/projects/${project.id}?starter=docker-image`
					: `/dashboard/projects/${project.id}`;
			// eslint-disable-next-line svelte/no-navigation-without-resolve
			await goto(target);
		} catch {
			error = 'Network error';
		} finally {
			busyStarter = null;
		}
	}

	type StarterCard = {
		id: Starter | 'coming-soon';
		title: string;
		description: string;
		icon?: IconSource;
		lucide?: typeof Container;
		enabled: boolean;
		onSelect?: () => void;
	};

	let cards = $derived<StarterCard[]>([
		{
			id: 'docker-image',
			title: 'Docker Image',
			description: 'Deploy a prebuilt image from any container registry.',
			lucide: Container,
			enabled: true,
			onSelect: () => launch('docker-image')
		},
		{
			id: 'empty-project',
			title: 'Empty Project',
			description: 'Start from scratch and add services later.',
			icon: Squares2x2,
			enabled: true,
			onSelect: () => launch('empty-project')
		},
		{
			id: 'coming-soon',
			title: 'GitHub Repository',
			description: 'Build and deploy from a Git repository.',
			icon: CodeBracket,
			enabled: false
		},
		{
			id: 'coming-soon',
			title: 'Database',
			description: 'Provision a managed database for your services.',
			icon: CircleStack,
			enabled: false
		},
		{
			id: 'coming-soon',
			title: 'Template',
			description: 'Launch a preconfigured stack from a template.',
			icon: RectangleStack,
			enabled: false
		}
	]);
</script>

<section
	class="mx-auto flex min-h-[70vh] w-full max-w-3xl flex-col items-center justify-start pt-12"
>
	<div class="mb-10 text-center">
		<h1 class="text-2xl font-semibold text-foreground">Start a new project</h1>
		<p class="mt-2 text-sm text-muted-foreground">
			Pick a starter to create your project. You can add more services and environments later.
		</p>
	</div>

	{#if !canEdit}
		<EmptyState
			icon={Server}
			title="You don't have permission to create projects"
			description="Ask a workspace owner or developer to create a project, or request a role change."
		>
			{#snippet actions()}
				<Button href="/dashboard/projects" variant="secondary" size="sm">Back to projects</Button>
			{/snippet}
		</EmptyState>
	{:else}
		<div class="grid w-full grid-cols-1 gap-3 sm:grid-cols-2">
			{#each cards as card, i (i)}
				{@const disabled = !card.enabled || busyStarter !== null}
				<button
					type="button"
					onclick={card.enabled ? card.onSelect : undefined}
					{disabled}
					class="group relative flex items-start gap-3 rounded-xl border border-border bg-surface p-4 text-left transition-all {card.enabled
						? 'hover:border-foreground/40 hover:shadow-sm disabled:cursor-not-allowed disabled:opacity-60'
						: 'cursor-not-allowed opacity-60'}"
				>
					<div class="mt-0.5 grid h-10 w-10 place-content-center rounded-lg bg-surface-muted">
						{#if card.lucide}
							{@const LucideIcon = card.lucide}
							<LucideIcon class="h-5 w-5 text-foreground" strokeWidth={1.75} />
						{:else if card.icon}
							<Icon src={card.icon} theme="outline" class="h-5 w-5 text-foreground" />
						{/if}
					</div>
					<div class="min-w-0 flex-1">
						<div class="flex items-center justify-between gap-2">
							<span class="font-medium text-foreground">{card.title}</span>
							{#if !card.enabled}
								<span
									class="rounded-md bg-surface-muted px-2 py-0.5 text-[10px] font-medium tracking-wide text-muted-foreground uppercase"
								>
									Coming soon
								</span>
							{:else if busyStarter === card.id}
								<svg
									class="h-4 w-4 animate-spin text-muted-foreground"
									xmlns="http://www.w3.org/2000/svg"
									fill="none"
									viewBox="0 0 24 24"
								>
									<circle
										class="opacity-25"
										cx="12"
										cy="12"
										r="10"
										stroke="currentColor"
										stroke-width="4"
									/>
									<path
										class="opacity-75"
										fill="currentColor"
										d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
									/>
								</svg>
							{/if}
						</div>
						<p class="mt-1 text-sm text-muted-foreground">{card.description}</p>
					</div>
				</button>
			{/each}
		</div>

		{#if error}
			<p class="mt-4 text-sm text-danger">{error}</p>
		{/if}

		<div class="mt-8">
			<Button href="/dashboard/projects" variant="ghost" size="sm">Cancel</Button>
		</div>
	{/if}
</section>
