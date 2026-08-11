<script lang="ts">
	import { page } from '$app/state';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { Cog6Tooth, Squares2x2 } from '@steeze-ui/heroicons';

	let { children } = $props();

	let projectId = $derived(page.params.id as string);
	let builderHref = $derived(`/projects/${projectId}`);
	let settingsHref = $derived(`/projects/${projectId}/settings`);
	let builderActive = $derived(page.url.pathname === builderHref);
	let settingsActive = $derived(page.url.pathname === settingsHref);
</script>

<div class="flex min-h-0 w-full flex-1">
	<nav
		class="mr-3 flex w-14 flex-none flex-col items-center gap-1 border-r border-border py-3 pr-3"
		aria-label="Project navigation"
	>
		<!-- eslint-disable svelte/no-navigation-without-resolve -->
		<a
			href={builderHref}
			aria-label="Builder"
			aria-current={builderActive ? 'page' : undefined}
			title="Builder"
			class="grid h-10 w-10 place-content-center rounded-xl text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none {builderActive
				? 'bg-muted text-foreground'
				: ''}"
		>
			<Icon src={Squares2x2} theme="outline" class="h-5 w-5" />
		</a>
		<a
			href={settingsHref}
			aria-label="Project settings"
			aria-current={settingsActive ? 'page' : undefined}
			title="Project settings"
			class="grid h-10 w-10 place-content-center rounded-xl text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none {settingsActive
				? 'bg-muted text-foreground'
				: ''}"
		>
			<Icon src={Cog6Tooth} theme="outline" class="h-5 w-5" />
		</a>
		<!-- eslint-enable svelte/no-navigation-without-resolve -->
	</nav>

	<div class="flex min-h-0 min-w-0 flex-1">
		{@render children()}
	</div>
</div>
