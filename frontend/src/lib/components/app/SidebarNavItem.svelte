<script lang="ts">
	type Props = {
		href: string;
		label: string;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		icon: any;
		active: boolean;
	};

	let { href, label, icon: Icon, active }: Props = $props();
</script>

<!-- eslint-disable svelte/no-navigation-without-resolve -->
<a
	{href}
	class="flex h-9 flex-row items-center rounded-[9px] transition-colors
		{active
		? 'bg-[#EEEFF1] text-[#242529]'
		: 'text-muted-foreground hover:bg-gray-100 hover:text-gray-950'}"
>
	<div class="grid h-9 w-9 flex-none place-content-center">
		<Icon size={16} strokeWidth={active ? 2 : 1.75} />
	</div>
	<span class="min-w-0 flex-1 truncate text-sm">{label}</span>
</a>
<!-- eslint-enable svelte/no-navigation-without-resolve -->
