<script lang="ts">
	import { page } from '$app/state';
	import { Squares2x2, Key, Server } from '@steeze-ui/heroicons';
	import SidebarNavItem from '$lib/components/app/SidebarNavItem.svelte';

	let { workspaceName }: { workspaceName: string } = $props();

	const navItems = [
		{ href: '/projects', label: 'Projects', icon: Squares2x2 },
		{ href: '/ssh-keys', label: 'SSH Keys', icon: Key },
		{ href: '/servers', label: 'Servers', icon: Server }
	];

	function isActive(href: string): boolean {
		return page.url.pathname.startsWith(href);
	}
</script>

<aside class="mb-4 flex w-48 flex-col bg-sidebar text-sidebar-foreground">
	<!-- Navigation -->
	<nav class="flex-1 overflow-y-auto px-2 pt-14 pb-2">
		<div class="flex flex-col gap-px">
			{#each navItems as item (item.href)}
				<SidebarNavItem
					href={item.href}
					label={item.label}
					icon={item.icon}
					active={isActive(item.href)}
				/>
			{/each}
		</div>
	</nav>

	<!-- Workspace -->
	<footer class="flex flex-col px-2">
		<div class="flex h-10 items-center select-none">
			<!-- eslint-disable svelte/no-navigation-without-resolve -->
			<a
				href="/projects"
				class="flex min-w-0 flex-1 flex-row items-center gap-2 rounded-md px-2.5 py-2"
			>
				<span
					class="flex h-5 w-5 flex-none items-center justify-center rounded-full bg-sidebar-primary text-[10px] font-bold text-sidebar-primary-foreground"
					>U</span
				>
				<span class="min-w-0 flex-1 truncate text-sm font-medium text-foreground"
					>{workspaceName}</span
				>
			</a>
			<!-- eslint-enable svelte/no-navigation-without-resolve -->
		</div>
	</footer>
</aside>
