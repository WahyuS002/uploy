<script lang="ts">
	import { page } from '$app/state';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { Squares2x2, Key, Server, ChevronUpDown } from '@steeze-ui/heroicons';
	import SidebarNavItem from '$lib/components/app/SidebarNavItem.svelte';

	let { workspaceName }: { workspaceName: string } = $props();

	const navItems = [
		{ href: '/dashboard/projects', label: 'Projects', icon: Squares2x2 },
		{ href: '/dashboard/ssh-keys', label: 'SSH Keys', icon: Key },
		{ href: '/dashboard/servers', label: 'Servers', icon: Server }
	];

	function isActive(href: string): boolean {
		return page.url.pathname.startsWith(href);
	}
</script>

<aside class="mb-4 flex w-48 flex-col bg-background">
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

	<!-- Workspace header -->
	<footer class="flex flex-col gap-1">
		<div class="flex flex-row px-2">
			<div class="flex h-10 w-full flex-row items-center">
				<div class="flex min-w-0 flex-1 items-center select-none">
					<!-- eslint-disable svelte/no-navigation-without-resolve -->
					<a
						href="/dashboard/projects"
						class="flex min-w-0 flex-1 flex-row items-center gap-2 rounded-md px-2.5 py-2"
					>
						<span
							class="flex h-5 w-5 flex-none items-center justify-center rounded-full bg-foreground text-[10px] font-bold text-surface"
							>U</span
						>
						<span class="min-w-0 flex-1 truncate text-sm font-medium text-foreground"
							>{workspaceName}</span
						>
					</a>
					<!-- eslint-enable svelte/no-navigation-without-resolve -->
					<button
						type="button"
						class="flex-none cursor-pointer rounded-md bg-transparent px-1.5 py-2 text-foreground hover:bg-gray-200"
					>
						<Icon src={ChevronUpDown} theme="outline" class="h-4 w-4" />
					</button>
				</div>
			</div>
		</div>
	</footer>
</aside>
