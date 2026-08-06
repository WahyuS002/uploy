<script lang="ts">
	import type { components } from '$lib/api/v1';
	import DeploymentLogs from '$lib/components/DeploymentLogs.svelte';
	import StatusBadge from '$lib/components/app/StatusBadge.svelte';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { XMark } from '@steeze-ui/heroicons';
	import { Container } from 'lucide-svelte';
	import { cn } from '$lib/components/ui/cn.js';
	import { formatDateTime, formatRelativeTime } from '$lib/format-date';
	import { quintOut } from 'svelte/easing';
	import type { TransitionConfig } from 'svelte/transition';

	type DeploymentResponse = components['schemas']['DeploymentResponse'];

	type Props = {
		deployment: DeploymentResponse;
		/** Context, not the subject — the deployment id is what this panel is about. */
		serviceName: string;
		onClose: () => void;
		/** Where it sits. The builder stacks it over the inspector; the service page
		 * has no panel to stack on and pins it to the viewport instead. */
		class?: string;
	};

	let { deployment, serviceName, onClose, class: className }: Props = $props();

	// Arrives from the top-right and travels down-left into place, while the panel
	// underneath recedes along the same diagonal — one gesture on one clock, which
	// is what says "a level deeper" rather than "a different screen". Entering
	// straight in from the right, the way the inspector itself arrives, would have
	// been the same motion twice at two depths.
	const stack: (node: Element) => TransitionConfig = () => ({
		duration: 260,
		easing: quintOut,
		// Its own width plus the gap it floats off the edge, so it starts fully
		// clear of the surface it covers; -24px is the height it falls from.
		css: (_t: number, u: number) =>
			`transform: translate3d(calc(${u} * (100% + 1rem)), ${u * -24}px, 0)`
	});
</script>

<aside
	transition:stack
	class={cn(
		'deployment-panel flex min-h-0 flex-col overflow-hidden rounded-xl border border-border bg-card text-card-foreground',
		className
	)}
	aria-label="Deployment {deployment.id.slice(0, 8)}"
>
	<!-- Same chip, padding and rhythm as the inspector header it covers, so the
	     two read as the same kind of surface one level apart. The service name is
	     demoted to a breadcrumb: you already know which service you are in, and
	     the id is the only thing that distinguishes this panel from the last one
	     you opened. -->
	<header class="flex flex-none items-center gap-3 border-b border-border px-5 py-4">
		<span class="grid h-9 w-9 flex-none place-content-center rounded-md bg-muted text-foreground">
			<Container class="h-4.5 w-4.5" strokeWidth={1.75} />
		</span>
		<div class="flex min-w-0 items-baseline gap-2">
			<span class="truncate text-[15px] text-muted-foreground">{serviceName}</span>
			<span class="flex-none text-muted-foreground/50" aria-hidden="true">/</span>
			<h2 class="flex-none font-mono text-xl font-semibold tracking-[-0.01em] text-foreground">
				{deployment.id.slice(0, 8)}
			</h2>
		</div>
		<StatusBadge status={deployment.status} class="flex-none" />
		<span
			class="ml-auto flex-none text-[13px] text-muted-foreground"
			title={formatDateTime(deployment.created_at)}
		>
			{formatRelativeTime(deployment.created_at)}
		</span>
		<button
			type="button"
			onclick={onClose}
			class="grid h-11 w-11 flex-none cursor-pointer place-content-center rounded-md text-muted-foreground hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-card focus-visible:outline-none md:h-8 md:w-8"
			aria-label="Close deployment logs"
		>
			<Icon src={XMark} theme="outline" class="h-4.5 w-4.5" />
		</button>
	</header>

	<!-- The output is the whole panel, so it takes the whole panel: it fills the
	     height and scrolls in place rather than sitting as a 288px slab with dead
	     space under it. -->
	<div class="min-h-0 flex-1 overflow-hidden p-5">
		<!-- Keyed: picking another deployment while the panel is open has to reopen
		     the stream, and the log component only connects on mount. -->
		{#key deployment.id}
			<DeploymentLogs deploymentId={deployment.id} fill />
		{/key}
	</div>
</aside>

<style>
	/* Floats over the panel it came from, so it takes the --shadow-float
	   exception to the hairline-only elevation rule. */
	.deployment-panel {
		box-shadow: var(--shadow-float);
	}
</style>
