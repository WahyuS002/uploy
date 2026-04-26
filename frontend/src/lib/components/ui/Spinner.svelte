<script lang="ts">
	import { cn } from './cn.js';

	type Props = {
		class?: string;
	};

	let { class: className }: Props = $props();
</script>

<span
	class={cn('inline-flex animate-spin', className)}
	aria-hidden="true"
	style="animation-timing-function: linear; animation-duration: 1.2s;"
>
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width="1em"
		height="1em"
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
	>
		<path
			d="M12 2v4m0 12v4M4.93 4.93l2.83 2.83m8.48 8.48 2.83 2.83M2 12h4m12 0h4M4.93 19.07l2.83-2.83m8.48-8.48 2.83-2.83"
		/>
	</svg>
</span>
