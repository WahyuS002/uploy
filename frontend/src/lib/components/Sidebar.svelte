<script lang="ts">
	import { page } from '$app/state';
	import { Squares2x2, Key, Server } from '@steeze-ui/heroicons';
	import SidebarNavItem from '$lib/components/app/SidebarNavItem.svelte';
	import SidebarWorkspaceMenu from '$lib/components/app/SidebarWorkspaceMenu.svelte';

	type Props = {
		workspaceName: string;
		workspaceRole?: string;
	};

	let { workspaceName, workspaceRole }: Props = $props();

	const navItems = [
		{ href: '/projects', label: 'Projects', icon: Squares2x2 },
		{ href: '/ssh-keys', label: 'SSH Keys', icon: Key },
		{ href: '/servers', label: 'Servers', icon: Server }
	];

	function isActive(href: string): boolean {
		return page.url.pathname.startsWith(href);
	}
</script>

<aside
	class="flex h-screen w-64 flex-none flex-col border-r border-border bg-background text-sidebar-foreground"
>
	<div class="flex h-14 flex-col justify-center border-b border-border px-2">
		<SidebarWorkspaceMenu {workspaceName} {workspaceRole} />
	</div>
	<nav class="flex-1 overflow-y-auto px-2 pt-3 pb-2">
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
</aside>
