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
	class="flex h-9 items-center gap-1.5 rounded-[9px] px-2 text-sm leading-5 transition-[background-color,color] duration-150 focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none
		{active
		? 'bg-sidebar-accent font-medium text-sidebar-accent-foreground'
		: 'font-medium text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground'}"
>
	<Icon src={icon} theme={active ? 'solid' : 'outline'} class="h-3.5 w-3.5 flex-none" />
	<span class="min-w-0 flex-1 truncate">{label}</span>
</a>
<!-- eslint-enable svelte/no-navigation-without-resolve -->
