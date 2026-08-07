<script lang="ts">
	import { page } from '$app/state';
	import AppTopbar from '$lib/components/AppTopbar.svelte';

	let { data, children } = $props();

	const tabs = [
		{ href: '/projects', label: 'Projects' },
		{ href: '/servers', label: 'Servers' }
	];

	let workspaceName = $derived(data.workspace?.name ?? 'Uploy');
	let initial = $derived(workspaceName.charAt(0).toUpperCase());
	let workspaceTitle = $derived(
		data.workspace?.role ? `${workspaceName} · ${data.workspace.role}` : workspaceName
	);
</script>

<div class="flex min-h-screen flex-col bg-background">
	<AppTopbar userEmail={data.user?.email ?? ''}>
		{#snippet leading()}
			<span class="flex min-w-0 items-center gap-2" title={workspaceTitle}>
				<span
					class="flex h-6 w-6 flex-none items-center justify-center rounded-md border border-brand-tint-edge bg-brand-tint text-[11px] font-semibold text-primary-deep"
				>
					{initial}
				</span>
				<span class="min-w-0 truncate text-sm font-medium text-foreground">{workspaceName}</span>
			</span>
		{/snippet}
	</AppTopbar>

	<!-- Nav moved off the sidebar: the underline is the only active marker, so the
	     tabs read as one row of section names rather than a second toolbar. -->
	<nav class="flex-none px-gutter">
		<div class="flex items-center gap-5">
			{#each tabs as tab (tab.href)}
				{@const active = page.url.pathname.startsWith(tab.href)}
				<!-- eslint-disable svelte/no-navigation-without-resolve -->
				<a
					href={tab.href}
					aria-current={active ? 'page' : undefined}
					class="border-b-2 py-2.5 text-sm transition-colors duration-150 outline-none focus-visible:ring-2 focus-visible:ring-ring/40
						{active
						? 'border-foreground font-medium text-foreground'
						: 'border-transparent text-muted-foreground hover:text-foreground'}"
				>
					{tab.label}
				</a>
				<!-- eslint-enable svelte/no-navigation-without-resolve -->
			{/each}
		</div>
	</nav>

	<!-- Same inset panel the builder's non-canvas routes use, so both halves of the
	     app frame their content identically. The canvas surface was tried here and
	     reverted: it only pays for itself at grid density, and a workspace with one
	     project reads as a small card adrift in a large empty field. -->
	<main
		class="mx-gutter mb-4 flex flex-1 flex-col rounded-lg border border-border bg-card px-4 py-8 text-card-foreground sm:px-6 sm:py-10"
	>
		<div class="mx-auto flex w-full max-w-5xl flex-1 flex-col">
			{@render children()}
		</div>
	</main>
</div>
