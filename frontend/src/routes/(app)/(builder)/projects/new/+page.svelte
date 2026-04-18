<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import Button from '$lib/components/ui/Button.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import { Icon, type IconSource } from '@steeze-ui/svelte-icon';
	import {
		Server,
		CodeBracket,
		CircleStack,
		RectangleStack,
		ChevronRight
	} from '@steeze-ui/heroicons';
	import { Archive, Container, Sparkles, SquareFunction, SquareTerminal } from 'lucide-svelte';

	type ProjectResponse = components['schemas']['ProjectResponse'];

	type Starter = 'empty-project' | 'docker-image';

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');

	let busyStarter = $state<Starter | null>(null);
	let error = $state('');
	let promptDraft = $state('');

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
					? `/projects/${project.id}?starter=docker-image`
					: `/projects/${project.id}`;
			// eslint-disable-next-line svelte/no-navigation-without-resolve
			await goto(target);
		} catch {
			error = 'Network error';
		} finally {
			busyStarter = null;
		}
	}

	type StarterRow = {
		id: string;
		title: string;
		icon?: IconSource;
		lucide?: typeof Container;
		interactive: boolean;
		showsChevron: boolean;
		starter?: Starter;
	};

	const rows: StarterRow[] = [
		{
			id: 'github',
			title: 'GitHub Repository',
			icon: CodeBracket,
			interactive: false,
			showsChevron: true
		},
		{
			id: 'database',
			title: 'Database',
			icon: CircleStack,
			interactive: false,
			showsChevron: true
		},
		{
			id: 'template',
			title: 'Template',
			icon: RectangleStack,
			interactive: false,
			showsChevron: true
		},
		{
			id: 'docker-image',
			title: 'Docker Image',
			lucide: Container,
			interactive: true,
			showsChevron: true,
			starter: 'docker-image'
		},
		{
			id: 'function',
			title: 'Function',
			lucide: SquareFunction,
			interactive: false,
			showsChevron: false
		},
		{
			id: 'bucket',
			title: 'Bucket',
			lucide: Archive,
			interactive: false,
			showsChevron: false
		},
		{
			id: 'empty-project',
			title: 'Empty Project',
			lucide: SquareTerminal,
			interactive: true,
			showsChevron: false,
			starter: 'empty-project'
		}
	];
</script>

<div class="mx-auto w-full max-w-[720px]">
	{#if !canEdit}
		<EmptyState
			icon={Server}
			title="You don't have permission to create projects"
			description="Ask a workspace owner or developer to create a project, or request a role change."
		>
			{#snippet actions()}
				<Button href="/projects" variant="secondary" size="sm">Back to projects</Button>
			{/snippet}
		</EmptyState>
	{:else}
		<section
			class="flex flex-col rounded-2xl border border-border bg-surface shadow-sm"
			aria-label="Start a new project"
		>
			<div class="p-3 sm:p-4">
				<Textarea
					bind:value={promptDraft}
					placeholder="What would you like to create?"
					rows={2}
					class="resize-none border-border-input bg-surface-muted/40 px-4 py-3 text-base shadow-none"
				/>
			</div>

			<div class="flex flex-col gap-0.5 px-3 pb-3 sm:px-4 sm:pb-4">
				<div
					class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm text-foreground"
					aria-disabled="true"
				>
					<Sparkles class="h-5 w-5 flex-none text-muted-foreground" strokeWidth={1.75} />
					<span>Create to-do list function with a database</span>
				</div>
				<div
					class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm text-muted-foreground opacity-60"
					aria-disabled="true"
				>
					<Sparkles class="h-5 w-5 flex-none" strokeWidth={1.75} />
					<span>Deploy Redis, Postgres, and a Bucket</span>
				</div>
			</div>

			<div class="border-t border-border"></div>

			<ul class="flex flex-col p-2">
				{#each rows as row (row.id)}
					{@const busy = row.starter != null && busyStarter === row.starter}
					{@const pending = busyStarter !== null}
					{#if row.interactive}
						<li>
							<button
								type="button"
								onclick={() => row.starter && launch(row.starter)}
								disabled={pending}
								class="flex w-full items-center gap-3 rounded-lg px-3 py-3 text-left text-sm text-foreground transition-colors hover:bg-surface-muted disabled:cursor-not-allowed disabled:hover:bg-transparent"
							>
								<span class="grid h-5 w-5 flex-none place-content-center">
									{#if row.lucide}
										{@const LucideIcon = row.lucide}
										<LucideIcon class="h-5 w-5" strokeWidth={1.75} />
									{:else if row.icon}
										<Icon src={row.icon} theme="outline" class="h-5 w-5" />
									{/if}
								</span>
								<span class="flex-1 truncate font-medium">{row.title}</span>
								{#if busy}
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
								{:else if row.showsChevron}
									<Icon
										src={ChevronRight}
										theme="outline"
										class="h-4 w-4 flex-none text-muted-foreground"
									/>
								{/if}
							</button>
						</li>
					{:else}
						<li
							class="flex w-full items-center gap-3 rounded-lg px-3 py-3 text-sm text-muted-foreground"
						>
							<span class="grid h-5 w-5 flex-none place-content-center">
								{#if row.lucide}
									{@const LucideIcon = row.lucide}
									<LucideIcon class="h-5 w-5" strokeWidth={1.75} />
								{:else if row.icon}
									<Icon src={row.icon} theme="outline" class="h-5 w-5" />
								{/if}
							</span>
							<span class="flex-1 truncate">{row.title}</span>
							{#if row.showsChevron}
								<Icon src={ChevronRight} theme="outline" class="h-4 w-4 flex-none" />
							{/if}
						</li>
					{/if}
				{/each}
			</ul>
		</section>

		{#if error}
			<p class="mt-3 text-sm text-danger">{error}</p>
		{/if}

		<div class="mt-4 flex justify-end">
			<Button href="/projects" variant="ghost" size="sm">Cancel</Button>
		</div>
	{/if}
</div>
