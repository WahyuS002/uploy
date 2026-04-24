<script lang="ts">
	import { Icon } from '@steeze-ui/svelte-icon';
	import { Bell, Cog6Tooth, ArrowRightOnRectangle, ArrowLeft } from '@steeze-ui/heroicons';
	import { DropdownMenu } from 'bits-ui';
	import { logout } from '$lib/auth/logout';

	let { userEmail, label }: { userEmail: string; label: string } = $props();

	let loggingOut = $state(false);
	let logoutError = $state('');

	async function onLogout(event: Event) {
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

<header class="flex h-14 w-full flex-none items-center justify-between gap-4 bg-chrome px-3">
	<div class="flex flex-none items-center gap-2">
		<!-- eslint-disable svelte/no-navigation-without-resolve -->
		<a
			href="/projects"
			class="flex flex-none items-center gap-2 rounded-md px-1 py-1.5 select-none"
		>
			<span
				class="flex h-7 w-7 items-center justify-center rounded-md bg-foreground text-xs font-bold text-surface"
				>U</span
			>
			<span class="text-sm font-semibold tracking-tight text-foreground">Uploy</span>
		</a>
		<!-- eslint-enable svelte/no-navigation-without-resolve -->
		<span class="ml-2 hidden h-4 w-px bg-border-chrome sm:block"></span>
		<span class="ml-2 hidden text-sm font-medium text-muted-foreground sm:inline">{label}</span>
	</div>

	<div class="flex flex-none items-center gap-1.5">
		<button
			type="button"
			title="Notifications"
			class="grid h-9 w-9 cursor-pointer place-content-center rounded-full text-muted-foreground hover:bg-chrome-hover hover:text-foreground"
		>
			<Icon src={Bell} theme="outline" class="h-[18px] w-[18px]" />
		</button>
		<button
			type="button"
			title="Settings"
			class="grid h-9 w-9 cursor-pointer place-content-center rounded-full text-muted-foreground hover:bg-chrome-hover hover:text-foreground"
		>
			<Icon src={Cog6Tooth} theme="outline" class="h-[18px] w-[18px]" />
		</button>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger
				title={userEmail}
				class="ml-1 grid h-9 w-9 flex-none cursor-pointer place-content-center rounded-full bg-chrome-active text-xs font-medium text-muted-foreground outline-none hover:bg-border-chrome hover:text-foreground focus-visible:ring-2 focus-visible:ring-foreground/40"
			>
				{userEmail.charAt(0).toUpperCase()}
			</DropdownMenu.Trigger>
			<DropdownMenu.Portal>
				<DropdownMenu.Content
					align="end"
					sideOffset={6}
					class="z-50 min-w-56 overflow-hidden rounded-lg border border-border bg-surface p-1 shadow-overlay"
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
						class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm text-foreground outline-none data-disabled:cursor-not-allowed data-disabled:opacity-60 data-highlighted:bg-surface-muted"
					>
						<Icon src={ArrowRightOnRectangle} theme="outline" class="h-3.5 w-3.5" />
						<span>{loggingOut ? 'Logging out...' : 'Logout'}</span>
					</DropdownMenu.Item>
					{#if logoutError}
						<p role="alert" class="px-2 pt-1 pb-1.5 text-xs text-danger">{logoutError}</p>
					{/if}
				</DropdownMenu.Content>
			</DropdownMenu.Portal>
		</DropdownMenu.Root>
	</div>
</header>
