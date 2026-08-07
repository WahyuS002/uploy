<script lang="ts">
	import { page } from '$app/state';
	import { Squares2x2, Server } from '@steeze-ui/heroicons';
	import SidebarNavItem from '$lib/components/app/SidebarNavItem.svelte';
	import SidebarWorkspaceMenu from '$lib/components/app/SidebarWorkspaceMenu.svelte';

	type Props = {
		workspaceName: string;
		workspaceRole?: string;
	};

	let { workspaceName, workspaceRole }: Props = $props();

	const navItems = [
		{ href: '/projects', label: 'Projects', icon: Squares2x2 },
		{ href: '/servers', label: 'Servers', icon: Server }
	];

	function isActive(href: string): boolean {
		return page.url.pathname.startsWith(href);
	}
</script>

<aside
	class="flex h-screen w-56 flex-none flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground"
>
	<div class="flex h-14 flex-col justify-center border-b border-sidebar-border px-2">
		<SidebarWorkspaceMenu {workspaceName} {workspaceRole} />
	</div>
	<nav class="flex-1 overflow-y-auto p-2">
		<div class="flex min-w-0 flex-col gap-1">
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
</aside>
