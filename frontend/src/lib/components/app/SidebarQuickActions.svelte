<script lang="ts">
	import QuickActionsDialog from './QuickActionsDialog.svelte';

	type Props = {
		workspaceRole?: string;
	};

	let { workspaceRole }: Props = $props();
	let open = $state(false);

	function isTextInputTarget(target: EventTarget | null): boolean {
		if (!(target instanceof HTMLElement)) return false;
		const tag = target.tagName;
		if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true;
		if (target.isContentEditable) return true;
		return false;
	}

	function onKeydown(e: KeyboardEvent) {
		if (e.defaultPrevented) return;

		if ((e.metaKey || e.ctrlKey) && !e.shiftKey && !e.altKey && e.key.toLowerCase() === 'k') {
			e.preventDefault();
			open = !open;
			return;
		}

		if (e.key === '/' && !e.metaKey && !e.ctrlKey && !e.altKey && !e.shiftKey) {
			if (open) return;
			if (isTextInputTarget(e.target)) return;
			e.preventDefault();
			open = true;
		}
	}
</script>

<svelte:window onkeydown={onKeydown} />

<div class="flex items-center gap-1.5 px-1 pb-2">
	<button
		type="button"
		onclick={() => (open = true)}
		class="group flex h-7 flex-1 items-center gap-2 rounded-lg border border-border bg-card px-2 text-left shadow-panel transition-colors duration-150 hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
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
			class="inline-flex h-[18px] items-center justify-center rounded border border-border bg-background px-1 font-sans text-[10px] font-medium text-muted-foreground"
		>
			⌘K
		</kbd>
	</button>
	<button
		type="button"
		onclick={() => (open = true)}
		aria-label="Open quick actions"
		class="flex h-7 w-7 flex-none items-center justify-center rounded-lg border border-border bg-card text-muted-foreground shadow-panel transition-colors duration-150 hover:bg-accent hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
	>
		<span class="text-[12px] font-medium">/</span>
	</button>
</div>

<QuickActionsDialog bind:open {workspaceRole} />
