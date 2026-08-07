<script lang="ts">
	import { cn } from './cn.js';

	type Props = {
		code: string;
		class?: string;
	};

	let { code, class: className }: Props = $props();
</script>

<pre
	class={cn(
		'overflow-x-auto rounded-lg bg-muted p-2 font-mono text-xs break-all whitespace-pre-wrap',
		className
	)}>{code}</pre>
