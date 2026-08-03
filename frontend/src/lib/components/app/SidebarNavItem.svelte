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
	class="flex h-8 items-center gap-2 overflow-hidden rounded-md px-1.5 text-[13px] leading-[18.5px] transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-sidebar-ring/40 focus-visible:outline-none
		{active
		? 'bg-foreground/[0.04] font-medium text-sidebar-accent-foreground'
		: 'font-[450] text-muted-foreground hover:bg-foreground/[0.02] hover:text-sidebar-accent-foreground active:bg-foreground/[0.04]'}"
>
	<Icon src={icon} theme="outline" class="h-5 w-5 flex-none" />
	<span class="min-w-0 flex-1 truncate">{label}</span>
</a>
<!-- eslint-enable svelte/no-navigation-without-resolve -->
