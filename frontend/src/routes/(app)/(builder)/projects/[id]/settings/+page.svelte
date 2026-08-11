<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import FormField from '$lib/components/app/FormField.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { toast } from '$lib/components/ui/toast/toast-service.svelte.js';
	import {
		Dialog,
		DialogContent,
		DialogFooter,
		DialogHeader,
		DialogTitle
	} from '$lib/components/ui/dialog';
	import { Cube } from '@steeze-ui/heroicons';

	type ProjectResponse = components['schemas']['ProjectResponse'];

	let { data }: { data: PageData } = $props();

	let projectId = $derived(page.params.id as string);
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');
	let isOwner = $derived(data.workspace?.role === 'owner');

	let project = $state<ProjectResponse | null>(null);
	let loaded = $state(false);
	let loadError = $state('');
	let name = $state('');
	let saveError = $state('');
	let saving = $state(false);
	let deleteOpen = $state(false);
	let deleteConfirm = $state('');
	let deleteError = $state('');
	let deleting = $state(false);

	let nameChanged = $derived(!!project && name.trim() !== project.name);
	let deleteConfirmed = $derived(!!project && deleteConfirm === project.name);

	$effect(() => {
		const id = projectId;
		let cancelled = false;

		project = null;
		loaded = false;
		loadError = '';
		name = '';

		(async () => {
			try {
				const { data: response, error } = await api.GET('/api/projects/{id}', {
					params: { path: { id } }
				});
				if (cancelled) return;
				if (error) {
					loadError = (error as { error?: string }).error ?? 'Failed to load project';
					return;
				}
				project = response ?? null;
				name = response?.name ?? '';
			} catch {
				if (!cancelled) loadError = 'Network error';
			} finally {
				if (!cancelled) loaded = true;
			}
		})();

		return () => {
			cancelled = true;
		};
	});

	async function saveProject() {
		if (!project || !canEdit || saving) return;
		const trimmedName = name.trim();
		if (!trimmedName) {
			saveError = 'Project name is required';
			return;
		}

		saveError = '';
		saving = true;
		try {
			const { data: updated, error } = await api.PUT('/api/projects/{id}', {
				params: { path: { id: projectId } },
				body: { name: trimmedName }
			});
			if (error) {
				saveError = (error as { error?: string }).error ?? 'Failed to update project';
				return;
			}
			if (!updated) return;
			project = updated;
			name = updated.name;
			toast.show({ tone: 'success', title: 'Project updated', duration: 3500 });
		} catch {
			saveError = 'Network error';
		} finally {
			saving = false;
		}
	}

	function openDeleteDialog() {
		deleteConfirm = '';
		deleteError = '';
		deleteOpen = true;
	}

	async function deleteProject() {
		if (!project || !isOwner || !deleteConfirmed || deleting) return;

		deleteError = '';
		deleting = true;
		try {
			const { error } = await api.DELETE('/api/projects/{id}', {
				params: { path: { id: projectId } }
			});
			if (error) {
				deleteError = (error as { error?: string }).error ?? 'Failed to delete project';
				return;
			}
			await goto('/projects', {
				state: { toastFlash: { tone: 'success', title: 'Project deleted' } }
			});
		} catch {
			deleteError = 'Network error';
		} finally {
			deleting = false;
		}
	}
</script>

<svelte:head>
	<title>{project ? `${project.name} settings · Uploy` : 'Project settings · Uploy'}</title>
</svelte:head>

<div class="min-h-0 flex-1 overflow-y-auto rounded-xl border border-border bg-card">
	<div class="mx-auto w-full max-w-3xl px-5 py-8 sm:px-8 sm:py-10">
		<header class="mb-8 max-w-2xl">
			<h1 class="text-2xl font-semibold tracking-[-0.02em] text-foreground">Project settings</h1>
			<p class="mt-2 text-sm leading-relaxed text-muted-foreground">
				Manage the project identity and actions that affect every environment inside it.
			</p>
		</header>

		{#if !loaded}
			<div class="overflow-hidden rounded-xl border border-border">
				<div class="space-y-3 px-5 py-5">
					<div class="h-4 w-24 animate-pulse rounded bg-muted"></div>
					<div class="h-3 w-64 max-w-full animate-pulse rounded bg-muted"></div>
					<div class="h-9 w-full animate-pulse rounded-lg bg-muted"></div>
				</div>
			</div>
		{:else if !project}
			<EmptyState
				icon={Cube}
				title="Project not found"
				description={loadError || 'It may have been deleted or you might not have access to it.'}
			>
				{#snippet actions()}
					<Button href="/projects" variant="secondary" size="sm">Back to projects</Button>
				{/snippet}
			</EmptyState>
		{:else}
			<div class="overflow-hidden rounded-xl border border-border">
				<section class="px-5 py-5">
					<div class="mb-5 max-w-xl">
						<h2 class="text-[15px] font-semibold text-foreground">General</h2>
						<p class="mt-1 text-[13px] leading-relaxed text-muted-foreground">
							The project name appears in navigation and the projects overview.
						</p>
					</div>

					<form
						onsubmit={(event) => {
							event.preventDefault();
							saveProject();
						}}
						class="grid max-w-xl gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end"
					>
						<FormField label="Project name">
							<Input bind:value={name} required disabled={!canEdit || saving} />
						</FormField>
						{#if canEdit}
							<Button
								type="submit"
								size="sm"
								loading={saving}
								disabled={!nameChanged || !name.trim()}
								class="w-full sm:w-auto"
							>
								{saving ? 'Saving...' : 'Save changes'}
							</Button>
						{/if}
					</form>

					{#if saveError}
						<p class="mt-2 text-sm text-destructive" role="alert">{saveError}</p>
					{/if}
				</section>

				{#if isOwner}
					<section
						class="flex flex-col gap-4 border-t border-border px-5 py-5 sm:flex-row sm:items-center sm:justify-between"
					>
						<div class="max-w-xl">
							<h2 class="text-[15px] font-semibold text-foreground">Delete project</h2>
							<p class="mt-1 text-[13px] leading-relaxed text-muted-foreground">
								Permanently removes this project, its environments, and all attached services.
							</p>
						</div>
						<Button
							type="button"
							variant="destructive"
							size="sm"
							class="w-full flex-none sm:w-auto"
							onclick={openDeleteDialog}
						>
							Delete project
						</Button>
					</section>
				{/if}
			</div>
		{/if}
	</div>
</div>

<Dialog bind:open={deleteOpen}>
	<DialogContent>
		<DialogHeader>
			<DialogTitle>Delete {project?.name ?? 'project'}?</DialogTitle>
		</DialogHeader>
		<div class="space-y-4 px-5 pb-5 text-sm text-muted-foreground">
			<p>
				All environments, services, domains, and variables in this project will be permanently
				deleted. This cannot be undone.
			</p>
			<FormField label={`Type ${project?.name ?? ''} to confirm`}>
				<Input bind:value={deleteConfirm} autocomplete="off" />
			</FormField>
			{#if deleteError}
				<p class="text-destructive" role="alert">{deleteError}</p>
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
				disabled={!deleteConfirmed}
				onclick={deleteProject}
			>
				{deleting ? 'Deleting...' : 'Delete project'}
			</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>
