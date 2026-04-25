<script lang="ts">
	import { Icon } from '@steeze-ui/svelte-icon';
	import { Bell, Cog6Tooth, ArrowRightOnRectangle } from '@steeze-ui/heroicons';
	import { DropdownMenu } from 'bits-ui';
	import { page } from '$app/state';
	import { logout } from '$lib/auth/logout';
	import { getDashboardTopbarMeta } from './topbar-meta';

	let { userEmail }: { userEmail: string } = $props();

	let meta = $derived(getDashboardTopbarMeta(page.route.id, page.data));

	let loggingOut = $state(false);
	let logoutError = $state('');

	async function onLogout(event: Event) {
		// Keep the menu open until the request resolves so the disabled state
		// is visible; Bits UI otherwise closes on select.
		event.preventDefault();
		if (loggingOut) return;
		loggingOut = true;
		logoutError = '';
		const result = await logout();
		if (!result.ok) {
			logoutError = result.message;
			loggingOut = false;
		}
	}
</script>

<header
	class="flex h-14 w-full flex-none items-center justify-between gap-4 border-b border-border bg-white px-4"
>
	<div class="flex min-w-0 flex-1 items-center gap-2">
		{#if meta.icon}
			<Icon src={meta.icon} theme="outline" class="h-4 w-4 flex-none text-muted-foreground" />
		{/if}
		{#if meta.title}
			<h1 class="truncate text-sm font-medium text-foreground">{meta.title}</h1>
		{/if}
	</div>

	<div class="flex flex-none items-center gap-1.5">
		<button
			type="button"
			title="Notifications"
			class="grid h-9 w-9 cursor-pointer place-content-center rounded-full text-muted-foreground hover:bg-accent hover:text-accent-foreground"
		>
			<Icon src={Bell} theme="outline" class="h-[18px] w-[18px]" />
		</button>
		<button
			type="button"
			title="Settings"
			class="grid h-9 w-9 cursor-pointer place-content-center rounded-full text-muted-foreground hover:bg-accent hover:text-accent-foreground"
		>
			<Icon src={Cog6Tooth} theme="outline" class="h-[18px] w-[18px]" />
		</button>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger
				title={userEmail}
				class="ml-1 grid h-9 w-9 flex-none cursor-pointer place-content-center rounded-full bg-accent text-xs font-medium text-accent-foreground outline-none hover:bg-accent/70 focus-visible:ring-2 focus-visible:ring-ring/40"
			>
				{userEmail.charAt(0).toUpperCase()}
			</DropdownMenu.Trigger>
			<DropdownMenu.Portal>
				<DropdownMenu.Content
					align="end"
					sideOffset={6}
					class="z-50 min-w-56 overflow-hidden rounded-lg border border-border bg-popover p-1 text-popover-foreground shadow-overlay"
				>
					<div
						class="truncate px-2 py-1.5 text-xs text-muted-foreground select-none"
						title={userEmail}
					>
						{userEmail || 'Signed in'}
					</div>
					<DropdownMenu.Separator class="my-1 h-px bg-border" />
					<DropdownMenu.Item
						onSelect={onLogout}
						disabled={loggingOut}
						class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm text-foreground outline-none data-disabled:cursor-not-allowed data-disabled:opacity-60 data-highlighted:bg-accent data-highlighted:text-accent-foreground"
					>
						<Icon src={ArrowRightOnRectangle} theme="outline" class="h-3.5 w-3.5" />
						<span>{loggingOut ? 'Logging out...' : 'Logout'}</span>
					</DropdownMenu.Item>
					{#if logoutError}
						<p role="alert" class="px-2 pt-1 pb-1.5 text-xs text-destructive">{logoutError}</p>
					{/if}
				</DropdownMenu.Content>
			</DropdownMenu.Portal>
		</DropdownMenu.Root>
	</div>
</header>
