<script lang="ts">
	import { Icon } from '@steeze-ui/svelte-icon';
	import { Bell, Cog6Tooth } from '@steeze-ui/heroicons';

	let { userEmail }: { userEmail: string } = $props();
</script>

<header
	class="flex h-14 w-full flex-none items-center justify-between gap-4 bg-background pr-4 pl-2"
>
	<a
		href="/dashboard/projects"
		class="flex flex-none items-center gap-2 rounded-md px-2 py-1.5 select-none"
	>
		<span
			class="flex h-7 w-7 items-center justify-center rounded-md bg-foreground text-xs font-bold text-surface"
			>U</span
		>
		<span class="text-sm font-semibold tracking-tight text-foreground">Uploy</span>
	</a>

	<div class="flex flex-none items-center gap-1.5">
		<button
			type="button"
			title="Notifications"
			class="grid h-9 w-9 cursor-pointer place-content-center rounded-full text-muted-foreground hover:bg-surface-muted hover:text-foreground"
		>
			<Icon src={Bell} theme="outline" class="h-[18px] w-[18px]" />
		</button>
		<button
			type="button"
			title="Settings"
			class="grid h-9 w-9 cursor-pointer place-content-center rounded-full text-muted-foreground hover:bg-surface-muted hover:text-foreground"
		>
			<Icon src={Cog6Tooth} theme="outline" class="h-[18px] w-[18px]" />
		</button>
		<button
			type="button"
			title={userEmail}
			class="ml-1 grid h-9 w-9 flex-none cursor-pointer place-content-center rounded-full bg-gray-300 text-xs font-medium text-gray-700 hover:bg-gray-400"
		>
			{userEmail.charAt(0).toUpperCase()}
		</button>
	</div>
</header>
