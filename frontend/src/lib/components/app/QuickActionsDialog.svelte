<script lang="ts">
	import { Command } from 'bits-ui';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { MagnifyingGlass } from '@steeze-ui/heroicons';
	import { goto } from '$app/navigation';
	import {
		Dialog,
		DialogContent,
		DialogTitle,
		DialogDescription
	} from '$lib/components/ui/dialog';
	import { QUICK_ACTIONS, filterByRole, groupActions, type QuickAction } from './quick-actions';

	type Props = {
		open: boolean;
		workspaceRole?: string;
	};

	let { open = $bindable(false), workspaceRole }: Props = $props();

	let visible = $derived(filterByRole(QUICK_ACTIONS, workspaceRole));
	let groups = $derived(groupActions(visible));
	let groupOrder: ReadonlyArray<keyof typeof groups> = ['Navigate', 'Create'] as const;

	function runAction(action: QuickAction) {
		open = false;
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		goto(action.href);
	}
</script>

<Dialog bind:open>
	<DialogContent
		showCloseButton={false}
		class="inset-y-auto top-[18vh] mt-0 mb-0 w-[min(92vw,720px)] max-w-none overflow-hidden rounded-2xl"
	>
		<DialogTitle class="sr-only">Quick actions</DialogTitle>
		<DialogDescription class="sr-only">
			Search and run navigation or create commands.
		</DialogDescription>

		<Command.Root loop label="Quick actions" class="flex flex-col">
			<div class="flex items-center gap-2 border-b border-border px-4">
				<Icon
					src={MagnifyingGlass}
					theme="outline"
					class="h-4 w-4 flex-none text-muted-foreground"
				/>
				<Command.Input
					placeholder="Search quick actions..."
					class="h-12 w-full bg-transparent text-sm text-foreground placeholder:text-muted-foreground focus:outline-none"
				/>
			</div>

			<Command.List class="max-h-90 overflow-y-auto p-2">
				<Command.Viewport>
					<Command.Empty class="px-3 py-8 text-center text-sm text-muted-foreground">
						No matching actions.
					</Command.Empty>

					{#each groupOrder as groupName (groupName)}
						{#if groups[groupName].length > 0}
							<Command.Group value={groupName} class="mb-1 last:mb-0">
								<Command.GroupHeading
									class="px-2 pt-2 pb-1 text-[11px] font-medium tracking-wide text-muted-foreground uppercase"
								>
									{groupName}
								</Command.GroupHeading>
								<Command.GroupItems>
									{#each groups[groupName] as action (action.id)}
										<Command.Item
											value={action.label}
											keywords={action.keywords}
											onSelect={() => runAction(action)}
											class="flex h-9 cursor-pointer items-center gap-2.5 rounded-md px-2 text-sm text-foreground outline-none data-[selected=true]:bg-accent data-[selected=true]:text-accent-foreground"
										>
											<Icon
												src={action.icon}
												theme="outline"
												class="h-4 w-4 flex-none text-muted-foreground"
											/>
											<span class="min-w-0 flex-1 truncate">{action.label}</span>
										</Command.Item>
									{/each}
								</Command.GroupItems>
							</Command.Group>
						{/if}
					{/each}
				</Command.Viewport>
			</Command.List>

			<div
				class="flex items-center justify-end gap-3 border-t border-border bg-muted/40 px-4 py-2 text-[11px] text-muted-foreground"
			>
				<span class="flex items-center gap-1">
					<kbd
						class="inline-flex h-5 min-w-5 items-center justify-center rounded border border-border bg-background px-1 font-sans text-[10px] text-foreground"
						>↑</kbd
					>
					<kbd
						class="inline-flex h-5 min-w-5 items-center justify-center rounded border border-border bg-background px-1 font-sans text-[10px] text-foreground"
						>↓</kbd
					>
					to navigate
				</span>
				<span class="flex items-center gap-1">
					<kbd
						class="inline-flex h-5 items-center justify-center rounded border border-border bg-background px-1.5 font-sans text-[10px] text-foreground"
						>Enter</kbd
					>
					to select
				</span>
				<span class="flex items-center gap-1">
					<kbd
						class="inline-flex h-5 items-center justify-center rounded border border-border bg-background px-1.5 font-sans text-[10px] text-foreground"
						>Esc</kbd
					>
					to close
				</span>
			</div>
		</Command.Root>
	</DialogContent>
</Dialog>
