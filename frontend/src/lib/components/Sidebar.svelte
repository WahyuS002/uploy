<script lang="ts">
	import { page } from '$app/state';
	import { LayoutGrid, Box, KeyRound, Server, LogOut, ChevronsUpDown } from 'lucide-svelte';

	let { workspaceName, userEmail }: { workspaceName: string; userEmail: string } = $props();

	const navItems = [
		{ href: '/dashboard/projects', label: 'Projects', icon: LayoutGrid },
		{ href: '/dashboard/services', label: 'Services', icon: Box },
		{ href: '/dashboard/ssh-keys', label: 'SSH Keys', icon: KeyRound },
		{ href: '/dashboard/servers', label: 'Servers', icon: Server }
	];

	function isActive(href: string): boolean {
		return page.url.pathname.startsWith(href);
	}

	async function handleLogout() {
		await fetch('/api/auth/logout', { method: 'POST' });
		window.location.href = '/login';
	}
</script>

<aside class="my-4 flex w-60 flex-col bg-[#FBFBFB]">
	<!-- Workspace header -->
	<header class="flex flex-col gap-1">
		<div class="flex flex-row px-2">
			<div class="flex h-10 w-full flex-row items-center">
				<div class="flex min-w-0 flex-1 items-center select-none">
					<a
						href="/dashboard/projects"
						class="flex min-w-0 flex-1 flex-row items-center gap-2 rounded-md px-2.5 py-2"
					>
						<span
							class="flex h-5 w-5 flex-none items-center justify-center rounded-full bg-black text-[10px] font-bold text-white"
							>U</span
						>
						<span class="min-w-0 flex-1 truncate text-sm font-medium">{workspaceName}</span>
					</a>
					<button
						type="button"
						class="flex-none cursor-pointer rounded-md bg-transparent px-1.5 py-2 text-gray-900 hover:bg-gray-200"
					>
						<ChevronsUpDown size={16} strokeWidth={1.5} />
					</button>
				</div>
			</div>
		</div>
	</header>

	<!-- Navigation -->
	<nav class="flex-1 overflow-y-auto px-2 py-2">
		<div class="flex flex-col gap-px">
			{#each navItems as item (item.href)}
				{@const active = isActive(item.href)}
				<a
					href={item.href}
					class="flex h-9 flex-row items-center rounded-[9px] transition-colors
						{active ? 'bg-[#EEEFF1] text-[#242529]' : 'text-gray-600 hover:bg-gray-100 hover:text-gray-950'}"
				>
					<div class="grid h-9 w-9 flex-none place-content-center">
						<item.icon size={16} strokeWidth={active ? 2 : 1.75} />
					</div>
					<span class="min-w-0 flex-1 truncate text-sm">{item.label}</span>
				</a>
			{/each}
		</div>
	</nav>

	<!-- User -->
	<section class="flex flex-col gap-0.5 px-2">
		<div class="flex h-9 w-full items-center gap-2 rounded-full pr-2 pl-2.5">
			<span
				class="flex h-5 w-5 flex-none items-center justify-center rounded-full bg-gray-300 text-[10px] font-medium text-gray-700"
			>
				{userEmail.charAt(0).toUpperCase()}
			</span>
			<span class="min-w-0 flex-1 truncate text-sm text-gray-950">
				{userEmail.split('@')[0]}
			</span>
			<button
				type="button"
				onclick={handleLogout}
				class="grid h-6 w-6 cursor-pointer place-content-center rounded-full border border-gray-300 bg-white text-gray-600 transition-colors hover:bg-gray-200 hover:text-gray-950"
				title="Sign out"
			>
				<LogOut size={12} strokeWidth={1.75} />
			</button>
		</div>
	</section>
</aside>
