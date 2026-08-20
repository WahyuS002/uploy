<script lang="ts" module>
	export type Starter = 'github-repo' | 'docker-image' | 'empty-project';

	export type StarterEnabled = Partial<Record<Starter, boolean>>;
</script>

<script lang="ts">
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ChevronRight } from '@steeze-ui/heroicons';
	import { Container, GitFork as Github, SquareTerminal } from 'lucide-svelte';

	type Props = {
		busyStarter?: Starter | null;
		enabled?: StarterEnabled;
		title?: string;
		onSelect: (starter: Starter) => void;
	};

	let {
		busyStarter = null,
		enabled = { 'github-repo': true, 'docker-image': true, 'empty-project': true },
		title = 'Start with',
		onSelect
	}: Props = $props();

	const headingId = $props.id();

	type StarterRow = {
		id: Starter;
		title: string;
		icon: typeof Container;
		// A chevron means "this opens a next step". Rows that commit immediately
		// don't get one, so the affordance stays honest.
		showsChevron: boolean;
	};

	const rows: StarterRow[] = [
		{ id: 'github-repo', title: 'GitHub Repository', icon: Github, showsChevron: true },
		{ id: 'docker-image', title: 'Docker Image', icon: Container, showsChevron: true },
		{ id: 'empty-project', title: 'Empty Project', icon: SquareTerminal, showsChevron: false }
	];

	// Unavailable starters are hidden, not greyed out. A menu of things you can't
	// pick is a roadmap, and a roadmap isn't a control. Whatever blocks a starter
	// explains itself in the node that owns the precondition.
	let visibleRows = $derived(rows.filter((row) => enabled[row.id] === true));
</script>

{#if visibleRows.length > 0}
	<section
		class="panel w-full overflow-hidden rounded-lg border border-border bg-card text-card-foreground"
		aria-labelledby={headingId}
	>
		<div class="border-b border-border/70 px-3 py-2">
			<h2 id={headingId} class="text-sm font-medium text-foreground">{title}</h2>
		</div>

		<!-- p-1 + rounded rows is the same geometry selectMenuItemVariants uses for every
		     dropdown in the app. A square full-bleed fill inside a rounded panel belongs
		     to a different family of control, and this list is a menu. The 4px inset plus
		     the row's px-2 lands the icon at 12px — flush with the header's px-3. -->
		<ul class="p-1">
			{#each visibleRows as row (row.id)}
				{@const busy = busyStarter === row.id}
				{@const pending = busyStarter !== null}
				{@const RowIcon = row.icon}
				{@const gridCols =
					row.showsChevron || busy ? 'grid-cols-[auto_1fr_auto]' : 'grid-cols-[auto_1fr]'}
				<li>
					<button
						type="button"
						onclick={() => onSelect(row.id)}
						disabled={pending}
						aria-busy={busy ? 'true' : undefined}
						class="grid w-full cursor-pointer items-center gap-x-3 rounded-md px-2 py-2 text-left text-sm text-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none focus-visible:ring-inset disabled:cursor-not-allowed disabled:hover:bg-transparent {pending &&
						!busy
							? 'opacity-50'
							: ''} {gridCols}"
					>
						<span class="text-muted-foreground">
							<RowIcon class="h-4 w-4" strokeWidth={1.75} />
						</span>
						<span class="truncate">{row.title}</span>
						{#if busy}
							<Spinner class="h-3.5 w-3.5 text-muted-foreground" />
						{:else if row.showsChevron}
							<Icon src={ChevronRight} theme="outline" class="h-3.5 w-3.5 text-muted-foreground" />
						{/if}
					</button>
				</li>
			{/each}
		</ul>
	</section>
{/if}

<style>
	.panel {
		box-shadow: var(--shadow-panel);
	}
</style>
