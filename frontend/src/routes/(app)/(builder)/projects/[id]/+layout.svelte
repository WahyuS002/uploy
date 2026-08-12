<script lang="ts">
	import { page } from '$app/state';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ChartBar, Cog6Tooth, Squares2x2 } from '@steeze-ui/heroicons';

	let { children } = $props();

	let projectId = $derived(page.params.id as string);
	let builderHref = $derived(`/projects/${projectId}`);
	let observabilityHref = $derived(`/projects/${projectId}/observability`);
	let settingsHref = $derived(`/projects/${projectId}/settings`);
	let builderActive = $derived(page.url.pathname === builderHref);
	let observabilityActive = $derived(page.url.pathname === observabilityHref);
	let settingsActive = $derived(page.url.pathname === settingsHref);
</script>

<div class="flex min-h-0 w-full flex-1 flex-col">
	<nav
		class="flex-none overflow-x-auto overflow-y-hidden border-b border-border"
		aria-label="Project sections"
	>
		<div class="flex min-w-max items-center gap-5 px-1">
			<!-- eslint-disable svelte/no-navigation-without-resolve -->
			<a
				href={builderHref}
				aria-current={builderActive ? 'page' : undefined}
				class="-mb-px inline-flex items-center gap-2 border-b-2 py-3 text-sm transition-colors duration-150 outline-none focus-visible:rounded-sm focus-visible:ring-2 focus-visible:ring-ring/40 {builderActive
					? 'border-foreground font-medium text-foreground'
					: 'border-transparent text-muted-foreground hover:text-foreground'}"
			>
				<Icon src={Squares2x2} theme="outline" class="h-4 w-4" />
				<span>Builder</span>
			</a>
			<a
				href={observabilityHref}
				aria-current={observabilityActive ? 'page' : undefined}
				class="-mb-px inline-flex items-center gap-2 border-b-2 py-3 text-sm transition-colors duration-150 outline-none focus-visible:rounded-sm focus-visible:ring-2 focus-visible:ring-ring/40 {observabilityActive
					? 'border-foreground font-medium text-foreground'
					: 'border-transparent text-muted-foreground hover:text-foreground'}"
			>
				<Icon src={ChartBar} theme="outline" class="h-4 w-4" />
				<span>Observability</span>
			</a>
			<a
				href={settingsHref}
				aria-current={settingsActive ? 'page' : undefined}
				class="-mb-px inline-flex items-center gap-2 border-b-2 py-3 text-sm transition-colors duration-150 outline-none focus-visible:rounded-sm focus-visible:ring-2 focus-visible:ring-ring/40 {settingsActive
					? 'border-foreground font-medium text-foreground'
					: 'border-transparent text-muted-foreground hover:text-foreground'}"
			>
				<Icon src={Cog6Tooth} theme="outline" class="h-4 w-4" />
				<span>Settings</span>
			</a>
			<!-- eslint-enable svelte/no-navigation-without-resolve -->
		</div>
	</nav>

	<div class="flex min-h-0 min-w-0 flex-1">
		{@render children()}
	</div>
</div>
