<script lang="ts" module>
	export type Starter = 'empty-project' | 'docker-image';

	type StarterId = 'github' | 'database' | 'template' | 'docker-image' | 'bucket' | 'empty-project';

	export type StarterEnabled = Partial<Record<Starter, boolean>>;
</script>

<script lang="ts">
	import Input from '$lib/components/ui/Input.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import { Icon, type IconSource } from '@steeze-ui/svelte-icon';
	import {
		CodeBracket,
		CircleStack,
		RectangleStack,
		ChevronRight,
		MagnifyingGlass
	} from '@steeze-ui/heroicons';
	import { Archive, Container, SquareTerminal } from 'lucide-svelte';

	type Props = {
		busyStarter?: Starter | null;
		enabled?: StarterEnabled;
		placeholder?: string;
		onSelect: (starter: Starter) => void;
	};

	let {
		busyStarter = null,
		enabled = { 'docker-image': true, 'empty-project': true },
		placeholder = 'What would you like to create?',
		onSelect
	}: Props = $props();

	let promptDraft = $state('');

	type StarterRow = {
		id: StarterId;
		title: string;
		icon?: IconSource;
		lucide?: typeof Container;
		showsChevron: boolean;
		starter?: Starter;
	};

	const rows: StarterRow[] = [
		{ id: 'github', title: 'GitHub Repository', icon: CodeBracket, showsChevron: true },
		{ id: 'database', title: 'Database', icon: CircleStack, showsChevron: true },
		{ id: 'template', title: 'Template', icon: RectangleStack, showsChevron: true },
		{
			id: 'docker-image',
			title: 'Docker Image',
			lucide: Container,
			showsChevron: true,
			starter: 'docker-image'
		},
		{ id: 'bucket', title: 'Bucket', lucide: Archive, showsChevron: false },
		{
			id: 'empty-project',
			title: 'Empty Project',
			lucide: SquareTerminal,
			showsChevron: false,
			starter: 'empty-project'
		}
	];

	function isInteractive(row: StarterRow): boolean {
		return row.starter != null && enabled[row.starter] === true;
	}

	let filteredRows = $derived.by(() => {
		const q = promptDraft.trim().toLowerCase();
		if (!q) return rows;
		return rows.filter((row) => row.title.toLowerCase().includes(q));
	});
</script>

<section
	class="panel overflow-hidden rounded-lg border border-gray-200 bg-card text-card-foreground"
	aria-label="Start a new resource"
>
	<div class="border-b border-border/70 px-2 py-1.5">
		<Input
			bind:value={promptDraft}
			{placeholder}
			class="border-transparent! bg-transparent! px-2 py-1.5 text-sm focus:shadow-none!"
		/>
	</div>

	<div class="py-1">
		{#if filteredRows.length === 0}
			<div class="flex items-center gap-2.5 px-4 py-3 text-sm text-muted-foreground">
				<Icon src={MagnifyingGlass} theme="outline" class="h-3.5 w-3.5 flex-none" />
				<span>No starters match "{promptDraft}"</span>
			</div>
		{:else}
			<ul>
				{#each filteredRows as row (row.id)}
					{@const interactive = isInteractive(row)}
					{@const busy = row.starter != null && busyStarter === row.starter}
					{@const pending = busyStarter !== null}
					{@const gridCols =
						row.showsChevron || !interactive || busy
							? 'grid-cols-[auto_1fr_auto]'
							: 'grid-cols-[auto_1fr]'}
					{#if interactive}
						<li>
							<button
								type="button"
								onclick={() => row.starter && onSelect(row.starter)}
								disabled={pending}
								class="grid w-full cursor-pointer items-center gap-x-3 px-3 py-2 text-left text-sm text-foreground transition-colors hover:bg-accent hover:text-accent-foreground disabled:cursor-not-allowed disabled:hover:bg-transparent {gridCols}"
							>
								<span class="text-muted-foreground/70">
									{#if row.lucide}
										{@const LucideIcon = row.lucide}
										<LucideIcon class="h-4 w-4" strokeWidth={1.75} />
									{:else if row.icon}
										<Icon src={row.icon} theme="outline" class="h-4 w-4" />
									{/if}
								</span>
								<span class="truncate">{row.title}</span>
								{#if busy}
									<Spinner class="h-3.5 w-3.5 text-muted-foreground/70" />
								{:else if row.showsChevron}
									<Icon
										src={ChevronRight}
										theme="outline"
										class="h-3.5 w-3.5 text-muted-foreground/70"
									/>
								{/if}
							</button>
						</li>
					{:else}
						<li
							class="grid cursor-default items-center gap-x-3 px-3 py-2 text-sm text-muted-foreground {gridCols}"
						>
							<span class="text-muted-foreground/70">
								{#if row.lucide}
									{@const LucideIcon = row.lucide}
									<LucideIcon class="h-4 w-4" strokeWidth={1.75} />
								{:else if row.icon}
									<Icon src={row.icon} theme="outline" class="h-4 w-4" />
								{/if}
							</span>
							<span class="truncate">{row.title}</span>
							<span
								class="rounded-sm bg-secondary px-1.5 py-0.5 text-[10px] font-medium tracking-wide text-muted-foreground/70 uppercase"
							>
								Soon
							</span>
						</li>
					{/if}
				{/each}
			</ul>
		{/if}
	</div>
</section>

<style>
	.panel {
		box-shadow: var(--shadow-panel);
	}
</style>
