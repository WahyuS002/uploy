<script lang="ts">
	import { Icon, type IconSource } from '@steeze-ui/svelte-icon';

	type Props = {
		href: string;
		label: string;
		icon: IconSource;
		active: boolean;
	};

	let { href, label, icon, active }: Props = $props();
</script>

<!-- eslint-disable svelte/no-navigation-without-resolve -->
<a
	{href}
	class="flex h-9 flex-row items-center rounded-[9px] transition-colors
		{active
		? 'bg-sidebar-accent text-sidebar-accent-foreground'
		: 'text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'}"
>
	<div class="grid h-9 w-9 flex-none place-content-center">
		<Icon src={icon} theme={active ? 'solid' : 'outline'} class="h-4 w-4" />
	</div>
	<span class="min-w-0 flex-1 truncate text-sm {active ? 'font-medium' : ''}">{label}</span>
</a>
<!-- eslint-enable svelte/no-navigation-without-resolve -->
