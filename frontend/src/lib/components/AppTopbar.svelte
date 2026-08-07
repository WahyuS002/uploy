<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { Bell, Cog6Tooth, ArrowRightOnRectangle } from '@steeze-ui/heroicons';
	import { DropdownMenu } from 'bits-ui';
	import { logout } from '$lib/auth/logout';

	type Props = {
		userEmail: string;
		label?: string;
		leading?: Snippet;
		action?: Snippet;
	};

	let { userEmail, label = '', leading, action }: Props = $props();

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

<!-- px-4 is the app's gutter: the tab row and the content panel both sit on it, so
     the topbar has to as well or the avatar overhangs the panel it sits above. -->
<header
	class="flex h-14 w-full flex-none items-center justify-between gap-4 bg-background px-gutter"
>
	<div class="flex min-w-0 flex-1 items-center gap-2">
		<!-- eslint-disable svelte/no-navigation-without-resolve -->
		<a href="/projects" class="flex flex-none items-center rounded-md py-1.5 select-none">
			<!-- The mark is transparent inside and out, so it sits on either theme as-is.
			     h-8 because the artwork carries its own margin — it reads at the same
			     weight the 28px lockup did. -->
			<img src="/logo.png" alt="Uploy" class="h-8 w-8" />
		</a>
		<!-- eslint-enable svelte/no-navigation-without-resolve -->
		<span class="ml-2 hidden h-4 w-px bg-border sm:block"></span>
		{#if leading}
			<div class="ml-2 flex min-w-0 flex-1 items-center gap-2">
				{@render leading()}
			</div>
		{:else if label}
			<span class="ml-2 hidden text-sm font-medium text-muted-foreground sm:inline">{label}</span>
		{/if}
	</div>

	<div class="flex flex-none items-center gap-1.5">
		{#if action}
			<div class="mr-1 flex items-center">
				{@render action()}
			</div>
		{/if}
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
				class="ml-1 grid h-9 w-9 flex-none cursor-pointer place-content-center rounded-full border border-brand-tint-edge bg-brand-tint text-sm font-medium text-primary-deep transition-colors duration-150 outline-none hover:border-primary-deep focus-visible:ring-2 focus-visible:ring-ring/40"
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
