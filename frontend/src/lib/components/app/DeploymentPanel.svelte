<script lang="ts">
	import type { components } from '$lib/api/v1';
	import DeploymentLogs from '$lib/components/DeploymentLogs.svelte';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { XMark } from '@steeze-ui/heroicons';
	import { CircleArrowUp, Clock, Container } from 'lucide-svelte';
	import { cn } from '$lib/components/ui/cn.js';
	import { formatDateTime, formatDuration, formatRelativeTime } from '$lib/format-date';
	import { quintOut } from 'svelte/easing';
	import type { TransitionConfig } from 'svelte/transition';

	type DeploymentResponse = components['schemas']['DeploymentResponse'];

	type Props = {
		deployment: DeploymentResponse;
		/** Context, not the subject — the deployment id is what this panel is about. */
		serviceName: string;
		onClose: () => void;
		/** The environment this deployment shipped to, where the caller knows it —
		 * the deployment itself only carries a service id. */
		environmentName?: string;
		/** Where it sits. The builder stacks it over the inspector; the service page
		 * has no panel to stack on and pins it to the viewport instead. */
		class?: string;
	};

	let { deployment, serviceName, onClose, environmentName, class: className }: Props = $props();

	// The deployment handed to this panel is a snapshot taken when it was opened;
	// the log stream underneath is the only thing that sees an in-progress run
	// finish. Written to on `done`, which a derived allows: it follows the prop
	// when another deployment is opened, and the stream overrides it until then.
	// Without it the details bar sits on "Building" until a reload.
	let status: string = $derived(deployment.status);

	// Owned by the log component — it is measured between the run's first and last
	// log line, which nothing out here can see.
	let elapsedSeconds = $state(0);

	const statusLabels: Record<string, string> = {
		success: 'Ready',
		failed: 'Error',
		in_progress: 'Building'
	};

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
		<button
			type="button"
			onclick={onClose}
			class="ml-auto grid h-11 w-11 flex-none cursor-pointer place-content-center rounded-md text-muted-foreground hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-card focus-visible:outline-none md:h-8 md:w-8"
			aria-label="Close deployment logs"
		>
			<Icon src={XMark} theme="outline" class="h-4.5 w-4.5" />
		</button>
	</header>

	<!-- The four facts you open a deployment for, before the output: when it ran,
	     how it ended, how long it took, where it went. Status and the timestamp
	     used to sit in the header above; they belong with the rest of the answer
	     rather than scattered across two rows saying the same thing twice.

	     A row that wraps rather than a fixed four-column grid: the panel is one
	     inspector wide on the builder and near-full width on the service page, and
	     four columns squeezed into 400px is four truncated columns. -->
	<dl class="flex flex-none flex-wrap gap-x-10 gap-y-4 border-b border-border px-5 py-4">
		<div class="min-w-0">
			<dt class="text-[13px] text-muted-foreground">Created</dt>
			<dd class="mt-1 truncate text-[15px] text-foreground">
				<time datetime={deployment.created_at}>{formatDateTime(deployment.created_at)}</time>
			</dd>
		</div>

		<div>
			<dt class="text-[13px] text-muted-foreground">Status</dt>
			<dd class="mt-1 flex items-center gap-2 text-[15px] text-foreground">
				<span
					class={cn(
						'h-2 w-2 flex-none rounded-full bg-muted-foreground',
						status === 'success' && 'bg-success-fill',
						status === 'failed' && 'bg-destructive',
						status === 'in_progress' && 'bg-warning motion-safe:animate-pulse'
					)}
					aria-hidden="true"
				></span>
				{statusLabels[status] ?? status}
			</dd>
		</div>

		<div>
			<dt class="text-[13px] text-muted-foreground">Duration</dt>
			<dd class="mt-1 flex items-center gap-2 text-[15px] text-foreground">
				<Clock class="h-4 w-4 flex-none text-muted-foreground" strokeWidth={1.75} />
				<span class="tabular-nums">{formatDuration(elapsedSeconds)}</span>
				<span class="text-muted-foreground">{formatRelativeTime(deployment.created_at)}</span>
			</dd>
		</div>

		{#if environmentName}
			<div class="min-w-0">
				<dt class="text-[13px] text-muted-foreground">Environment</dt>
				<dd class="mt-1 flex items-center gap-2 text-[15px] text-foreground">
					<CircleArrowUp class="h-4 w-4 flex-none text-muted-foreground" strokeWidth={1.75} />
					<span class="truncate">{environmentName}</span>
				</dd>
			</div>
		{/if}
	</dl>

	<!-- The output is the whole panel, so it takes the whole panel: it fills the
	     height and scrolls in place rather than sitting as a 288px slab with dead
	     space under it. -->
	<div class="min-h-0 flex-1 overflow-hidden p-5">
		<!-- No {#key} any more: the log component reopens its own stream when the id
		     changes, so forcing a remount here was working around something that is
		     now handled where it belongs.

		     This panel only ever shows deployments that already ran, so the status
		     always arrives with the id — which is what stops a two-day-old run from
		     replaying as though it were happening. -->
		<DeploymentLogs
			deploymentId={deployment.id}
			deploymentStatus={deployment.status}
			bind:elapsedSeconds
			onDone={(s) => (status = s)}
			fill
		/>
	</div>
</aside>

<style>
	/* Floats over the panel it came from, so it takes the --shadow-float
	   exception to the hairline-only elevation rule. */
	.deployment-panel {
		box-shadow: var(--shadow-float);
	}
</style>
