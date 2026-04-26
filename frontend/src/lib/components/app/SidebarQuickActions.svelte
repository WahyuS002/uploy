<script lang="ts">
	import QuickActionsDialog from './QuickActionsDialog.svelte';

	type Props = {
		workspaceRole?: string;
	};

	let { workspaceRole }: Props = $props();
	let open = $state(false);

	function onKeydown(e: KeyboardEvent) {
		if (e.defaultPrevented) return;
		if ((e.metaKey || e.ctrlKey) && !e.shiftKey && !e.altKey && e.key.toLowerCase() === 'k') {
			e.preventDefault();
			open = !open;
		}
	}
</script>

<svelte:window onkeydown={onKeydown} />

<div class="px-1 pb-2">
	<button
		type="button"
		onclick={() => (open = true)}
		class="group flex h-7 w-full items-center gap-2 rounded-lg border border-border bg-card px-2 text-left shadow-panel transition-colors duration-150 hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
	>
		<svg
			class="h-3.5 w-3.5 flex-none text-muted-foreground"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
			aria-hidden="true"
		>
			<circle cx="11" cy="11" r="7" />
			<path d="m20 20-3.5-3.5" />
		</svg>
		<span class="min-w-0 flex-1 truncate text-[13px] font-medium text-muted-foreground">
			Quick actions
		</span>
		<kbd
			class="inline-flex h-4.5 items-center justify-center rounded border border-border bg-background px-1 font-sans text-[10px] font-medium text-muted-foreground"
		>
			⌘K
		</kbd>
	</button>
</div>

<QuickActionsDialog bind:open {workspaceRole} />
