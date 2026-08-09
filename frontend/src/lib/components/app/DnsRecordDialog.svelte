<script lang="ts">
	import Button from '$lib/components/ui/Button.svelte';
	import { cn } from '$lib/components/ui/cn.js';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { Check, CheckCircle, ClipboardDocument, ExclamationTriangle } from '@steeze-ui/heroicons';
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogFooter,
		DialogTitle
	} from '$lib/components/ui/dialog';
	import { scale } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';
	import { countdownTo } from '$lib/countdown.svelte';
	import type { components } from '$lib/api/v1';

	type ServiceDomainResponse = components['schemas']['ServiceDomainResponse'];

	type Props = {
		/** The domain to explain, or null when nothing is being explained. */
		domain: ServiceDomainResponse | null;
		/** Whatever the server was registered as — an IP, or a hostname. */
		serverHost?: string | null;
		/**
		 * The proxy does not know this hostname yet, because the running container
		 * predates it. Until that is fixed the record cannot be validated, so this
		 * is the step the dialog leads with.
		 */
		needsDeploy?: boolean;
		onDeploy?: () => void;
		onClose: () => void;
	};

	let { domain, serverHost, needsDeploy = false, onDeploy, onClose }: Props = $props();

	/**
	 * The mark next to the record. It reads the domain's real status rather than
	 * being drawn on as decoration: a warning triangle that stays a warning after
	 * the record is correct teaches people to stop reading it.
	 *
	 * The status comes from the TLS reconciler, which checks every 60s whether a
	 * certificate exists for this hostname — and one cannot exist unless the
	 * record resolves to this server. So `ready` is proof the DNS is right, and
	 * anything else is honestly "not confirmed yet".
	 */
	const marks = {
		pending: {
			icon: ExclamationTriangle,
			class: 'text-warning',
			label: 'Not detected yet'
		},
		ready: {
			icon: CheckCircle,
			class: 'text-success',
			label: 'Verified'
		},
		error: {
			icon: ExclamationTriangle,
			class: 'text-destructive',
			label: 'Last check failed'
		}
	};
	// The undeployed case outranks the status, because it explains it: with no
	// router for this hostname no certificate is ever requested, so `pending` is
	// the symptom and this is the cause.
	const mark = $derived(
		needsDeploy && domain?.status !== 'ready'
			? { icon: ExclamationTriangle, class: 'text-warning', label: 'Not deployed yet' }
			: (marks[domain?.status ?? 'pending'] ?? marks.pending)
	);

	// An A record wants an address, and `host` is free text: somebody typed
	// either. A hostname is not wrong to have registered — SSH resolves it fine —
	// it just cannot be the right-hand side of an A record.
	const ipv4 = /^\d{1,3}(\.\d{1,3}){3}$/;
	const isAddress = $derived(!!serverHost && (ipv4.test(serverHost) || serverHost.includes(':')));

	const recordType = $derived(serverHost?.includes(':') ? 'AAAA' : 'A');

	/**
	 * The name a DNS provider wants in its "Name"/"Host" field: the part of the
	 * domain below the zone, or @ for the zone itself.
	 *
	 * ponytail: counts labels, so it reads `example.co.uk` as a subdomain of
	 * `co.uk` and offers `example` where the answer is `@`. Fixing it properly
	 * means shipping the public suffix list, which is a lot of bytes for a hint —
	 * the full domain is printed underneath so a wrong guess is visible rather
	 * than silent. Swap in a PSL lookup if multi-part TLDs turn up in practice.
	 */
	const recordName = $derived.by(() => {
		const labels = (domain?.domain ?? '').split('.');
		return labels.length > 2 ? labels.slice(0, -2).join('.') : '@';
	});

	// The API stops sending this once the domain is settled, which is exactly when
	// there is no longer anything to wait for — so the countdown disappears on its
	// own rather than needing a condition here.
	const nextCheck = countdownTo(() => domain?.next_check_at);

	// Which field was just copied, so only that cell acknowledges it. Keyed by
	// column name rather than by value: `A` and `@` are never equal, but a record
	// whose name happened to match its value would otherwise tick both.
	let copied = $state('');

	async function copyCell(field: string, text: string) {
		await navigator.clipboard.writeText(text);
		copied = field;
		// Guarded: copying a second cell inside the two seconds would otherwise let
		// the first one's timer clear the new tick.
		setTimeout(() => {
			if (copied === field) copied = '';
		}, 2000);
	}
</script>

<!-- Each field is its own copy target, because a DNS form is filled one field at
     a time and a single button for the whole row copies the wrong two-thirds.

     The icon stays visible rather than appearing on hover: a target you cannot
     see is not a target on a touchscreen, and dimmed carries the same "this is
     secondary" without hiding it. -->
{#snippet field(name: string, text: string)}
	<!-- The padding is on the cell and the highlight is inset inside it, so the
	     hovered surface stops short of its neighbours instead of running edge to
	     edge into them. 4px here plus 8px on the button still lands the text on
	     the header's 12px, so nothing shifts when the surface appears. -->
	<td class="p-1 align-top">
		<button
			type="button"
			onclick={() => copyCell(name, text)}
			aria-label="Copy {name.toLowerCase()} {text}"
			class="group flex w-full cursor-pointer items-center gap-3 rounded-md px-2 py-1.5 text-left transition-colors hover:bg-accent focus-visible:bg-accent focus-visible:outline-none"
		>
			<span class="min-w-0 flex-1 font-mono break-all text-foreground">{text}</span>
			<Icon
				src={copied === name ? Check : ClipboardDocument}
				theme="outline"
				class={cn(
					'h-3.5 w-3.5 flex-none transition-colors',
					copied === name ? 'text-success' : 'text-muted-foreground/40 group-hover:text-foreground'
				)}
			/>
		</button>
	</td>
{/snippet}

<Dialog open={domain !== null} onOpenChange={(open) => !open && onClose()}>
	<DialogContent class="w-[min(92vw,34rem)] max-w-none overflow-hidden">
		<DialogHeader class="px-5 pt-4 pr-12 pb-1">
			<DialogTitle>Configure DNS</DialogTitle>
		</DialogHeader>

		<div class="px-5 pt-2 pb-5">
			<p class="text-[13px] text-muted-foreground">
				Add this DNS record at your domain provider for
				<span class="font-mono text-foreground">{domain?.domain}</span>.
			</p>

			<!-- A table for one row, because the column headings are the instruction:
			     every DNS provider asks for these three fields under these names, and
			     a prose sentence would make the reader do the mapping themselves. -->
			<div class="mt-3 overflow-hidden rounded-lg border border-border">
				<table class="w-full text-left text-[13px]">
					<thead>
						<tr class="border-b border-border text-xs text-muted-foreground">
							<th scope="col" class="w-0 py-2 pr-1 pl-3"><span class="sr-only">Status</span></th>
							<th scope="col" class="px-3 py-2 font-medium">Type</th>
							<th scope="col" class="px-3 py-2 font-medium">Name</th>
							<th scope="col" class="px-3 py-2 font-medium">Value</th>
						</tr>
					</thead>
					<tbody>
						<tr>
							<!-- Its own cell, not tucked inside the Type button: the mark is a
							     reading, and putting it in the copy target would mean clicking it
							     copies something. -->
							<td class="w-0 py-2.5 pr-1 pl-3 align-top">
								<!-- Keyed, so the amber triangle becoming a green tick is a moment
								     rather than a swap nobody catches. This is the one state change
								     the whole dialog exists for; 180ms is long enough to register
								     and short enough that it is never in the way. -->
								{#key domain?.status}
									<span class="block" in:scale={{ duration: 180, start: 0.7, easing: quintOut }}>
										<Icon
											src={mark.icon}
											theme="outline"
											class={cn('h-4 w-4', mark.class)}
											aria-hidden="true"
										/>
									</span>
								{/key}
								<span class="sr-only">{mark.label}</span>
							</td>
							{@render field('Type', recordType)}
							{@render field('Name', recordName)}
							{#if isAddress}
								{@render field('Value', serverHost ?? '')}
							{:else}
								<!-- Not a copy target: there is nothing here worth pasting. A
								     hostname would fail the moment it landed in an A record, so
								     this says what to go and find instead. -->
								<!-- Same 12px inset as a copyable cell, reached without the nesting
							     since there is no surface to hold off the edges here. -->
								<td class="px-3 py-2.5 align-top text-muted-foreground">
									your server's IP address{serverHost ? ` (Uploy reaches it at ${serverHost})` : ''}
								</td>
							{/if}
						</tr>
					</tbody>
				</table>
			</div>

			<!-- The mark's meaning in words, because a triangle on its own is a mood,
			     not a message — and the person who most needs this sentence is the one
			     who cannot tell amber from green. -->
			<p class="mt-3 flex items-start gap-2 text-[13px] text-muted-foreground">
				<!-- Nothing is happening yet while the deploy is outstanding, so nothing
				     breathes. A dot that pulsed here would be claiming progress on a check
				     that cannot run. -->
				{#if domain && domain.status !== 'ready' && !needsDeploy}
					<!-- Breathing, because it is genuinely doing something: this panel
					     re-reads the domain on the reconciler's schedule and will change
					     under you. A still dot would be a label; this is a state. -->
					<span class={cn('watch-dot', mark.class)} aria-hidden="true"></span>
				{/if}
				<span class="min-w-0">
					<span class={cn('font-medium', mark.class)}>{mark.label}.</span>
					{#if domain?.status === 'ready'}
						DNS record is active.
					{:else if needsDeploy}
						Deploy your app so Uploy can configure proxy routing and verify DNS.
					{:else if nextCheck.seconds !== null}
						<!-- The number counts to the reconciler's next pass, which is the only
						     moment this can change — not to a refresh of our own, which would
						     run out four times as often and mean nothing three of them.

						     It ends the sentence rather than sitting inside it. "Checking again
						     in 60s." narrowing to "9s." changes this text's width every time it
						     crosses a digit, and the words that used to follow it slid along the
						     line once a second — far enough, near the container edge, to re-wrap
						     the paragraph. Nothing follows it now, so nothing moves.

						     "Updates automatically" went with the move, and was not replaced:
						     a countdown running down in front of the reader is the same promise,
						     demonstrated instead of asserted. It still gets said to a screen
						     reader, which cannot see the number tick.

						     aria-hidden because a digit read aloud every second is unusable;
						     the status text before it carries the state. -->
						<span class="whitespace-nowrap tabular-nums" aria-hidden="true">
							{nextCheck.seconds > 0 ? `Checking again in ${nextCheck.seconds}s.` : 'Checking now…'}
						</span>
						<span class="sr-only">Uploy re-checks this automatically.</span>
					{:else}
						Updates automatically.
					{/if}
				</span>
			</p>

			{#if domain?.status === 'error' && domain.last_error}
				<p class="mt-1.5 text-[13px] break-words text-destructive">{domain.last_error}</p>
			{/if}
		</div>

		<DialogFooter>
			<!-- The action sits with the instruction that asks for it. Sending the
			     reader to another tab to find a Deploy button — or stacking a second
			     modal on top of this one — would be two steps for one decision. -->
			{#if needsDeploy && onDeploy}
				<Button type="button" variant="secondary" size="sm" onclick={onClose}>Later</Button>
				<Button type="button" size="sm" onclick={onDeploy}>Deploy now</Button>
			{:else}
				<Button type="button" size="sm" onclick={onClose}>Done</Button>
			{/if}
		</DialogFooter>
	</DialogContent>
</Dialog>

<style>
	/* Same vocabulary as the log panels' status mark: in this app a dot that
	   breathes means "still happening", and it should mean that everywhere.
	   Colour comes from the caller so the dot agrees with the triangle above it.

	   No motion-reduce guard, matching the app's stance in layout.css: this is a
	   fade, not movement, and the variants are reserved for things that travel. */
	.watch-dot {
		flex: none;
		/* Optical, not metric: 0.45rem down puts a 6px dot on the x-height of the
		   13px line beside it rather than on its box. */
		margin-top: 0.45rem;
		width: 0.375rem;
		height: 0.375rem;
		border-radius: 50%;
		background: currentColor;
		animation: watch-pulse 1.6s ease-in-out infinite;
	}

	@keyframes watch-pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.3;
		}
	}
</style>
