<script lang="ts">
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ChevronDown, Squares2x2, Server, Key } from '@steeze-ui/heroicons';
	import { DropdownMenu } from 'bits-ui';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	type Props = {
		workspaceName: string;
		workspaceRole?: string;
	};

	let { workspaceName, workspaceRole }: Props = $props();

	let initial = $derived((workspaceName?.charAt(0) ?? 'W').toUpperCase());

	const links = [
		{ href: '/projects', label: 'Projects', icon: Squares2x2 },
		{ href: '/servers', label: 'Servers', icon: Server },
		{ href: '/ssh-keys', label: 'SSH Keys', icon: Key }
	] as const;

	function formatRole(role?: string): string {
		if (!role) return '';
		return role.charAt(0).toUpperCase() + role.slice(1);
	}
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger
		class="group flex h-10 w-full cursor-pointer items-center gap-2 rounded-md px-2 text-left outline-none hover:bg-sidebar-accent focus-visible:ring-2 focus-visible:ring-sidebar-ring/40"
	>
		<span
			class="flex h-6 w-6 flex-none items-center justify-center rounded-md bg-sidebar-primary text-[11px] font-semibold text-sidebar-primary-foreground"
		>
			{initial}
		</span>
		<span class="min-w-0 flex-1 truncate text-sm font-medium text-foreground">
			{workspaceName}
		</span>
		<Icon
			src={ChevronDown}
			theme="outline"
			class="h-3.5 w-3.5 flex-none text-muted-foreground transition-transform group-data-[state=open]:rotate-180"
		/>
	</DropdownMenu.Trigger>
	<DropdownMenu.Portal>
		<DropdownMenu.Content
			align="start"
			sideOffset={6}
			class="z-50 min-w-60 overflow-hidden rounded-lg border border-border bg-popover p-1 text-popover-foreground shadow-overlay"
		>
			<div class="flex items-center gap-2 px-2 py-2 select-none">
				<span
					class="flex h-7 w-7 flex-none items-center justify-center rounded-md bg-sidebar-primary text-xs font-semibold text-sidebar-primary-foreground"
				>
					{initial}
				</span>
				<div class="min-w-0 flex-1">
					<div class="truncate text-sm font-medium text-foreground">{workspaceName}</div>
					{#if workspaceRole}
						<div class="truncate text-xs text-muted-foreground">{formatRole(workspaceRole)}</div>
					{/if}
				</div>
			</div>
			<DropdownMenu.Separator class="my-1 h-px bg-border" />
			{#each links as link (link.href)}
				<DropdownMenu.Item
					onSelect={() => goto(resolve(link.href))}
					class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm text-foreground outline-none data-highlighted:bg-accent data-highlighted:text-accent-foreground"
				>
					<Icon src={link.icon} theme="outline" class="h-3.5 w-3.5 text-muted-foreground" />
					<span>{link.label}</span>
				</DropdownMenu.Item>
			{/each}
		</DropdownMenu.Content>
	</DropdownMenu.Portal>
</DropdownMenu.Root>
