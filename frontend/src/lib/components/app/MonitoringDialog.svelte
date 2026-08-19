<script lang="ts">
	import { tick } from 'svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import { Button, Input, ToggleGroup, toast } from '$lib/components/ui';
	import { Dialog, DialogContent, DialogHeader, DialogTitle } from '$lib/components/ui/dialog';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ChevronRight, Pencil } from '@steeze-ui/heroicons';
	import FormField from './FormField.svelte';

	type ServerResponse = components['schemas']['ServerResponse'];

	type Props = {
		/** The server to configure, or null when the dialog is closed. */
		serverId: string | null;
		onClose: () => void;
		/** Called after the agent is provisioned, so the caller can refresh whatever
		    view it was showing. */
		onSaved?: () => void | Promise<void>;
	};

	let { serverId, onClose, onSaved }: Props = $props();

	// The windows worth offering. The API accepts 1 to 30, but a free-text day count
	// is a question nobody arrives with an answer to; these four cover the reasons
	// people actually keep metrics, and a server already set to something else keeps
	// its value as a fifth choice.
	const RETENTION_PRESETS = [3, 7, 14, 30];

	let server = $state<ServerResponse | null>(null);
	let loadError = $state('');
	let saving = $state(false);
	let portLocked = $state(true);
	let portInput = $state<HTMLInputElement | null>(null);
	let fqdnInput = $state<HTMLInputElement | null>(null);
	let exposePublicly = $state(false);
	let port = $state('9184');
	let retentionDays = $state('7');
	let fqdn = $state('');

	let retentionOptions = $derived.by(() => {
		const current = Number(retentionDays);
		const days =
			Number.isInteger(current) && current >= 1 && current <= 30
				? [...new Set([...RETENTION_PRESETS, current])].sort((a, b) => a - b)
				: RETENTION_PRESETS;
		return days.map((day) => ({ value: String(day), label: `${day} days` }));
	});

	// Loaded rather than passed in: the observability page knows only which server is
	// unmonitored, not its host or current monitoring settings, and one request when a
	// modal opens is cheaper than widening the summary shape it does have.
	$effect(() => {
		const id = serverId;
		let cancelled = false;
		server = null;
		loadError = '';
		if (!id) return;
		void (async () => {
			try {
				const { data, error } = await api.GET('/api/servers');
				if (cancelled) return;
				const found = data?.find((candidate) => candidate.id === id);
				if (error || !found) {
					loadError = 'This server is no longer available.';
					return;
				}
				server = found;
				seed(found);
			} catch {
				if (!cancelled) loadError = 'Could not load this server.';
			}
		})();
		return () => {
			cancelled = true;
		};
	});

	function seed(target: ServerResponse) {
		portLocked = true;
		port = String(target.monitoring.port || 9184);
		retentionDays = String(target.monitoring.retention_days || 7);
		lastRetention = retentionDays;
		fqdn = target.monitoring.fqdn ?? '';
		exposePublicly = fqdn !== '';
	}

	async function editPort() {
		portLocked = false;
		await tick();
		portInput?.focus();
		portInput?.select();
	}

	async function revealDomain() {
		if (!exposePublicly) return;
		await tick();
		fqdnInput?.focus();
	}

	// A toggle group deselects when its active item is clicked again. Retention has no
	// "none", so the window that was there stands rather than emptying the control.
	let lastRetention = '7';

	function holdRetention(next: string) {
		if (next) lastRetention = next;
		else retentionDays = lastRetention;
	}

	async function save() {
		if (!server || saving) return;
		const portValue = Number(port);
		const retentionValue = Number(retentionDays);
		// Unchecked means unpublished, whatever is still sitting in the field: the box is
		// the answer, and the domain is only how that answer gets carried out.
		const domain = exposePublicly ? fqdn.trim() : '';
		if (exposePublicly && !domain) {
			toast.error({
				title: 'Domain required',
				description: 'Enter the domain to publish on, or clear the checkbox.'
			});
			return;
		}
		if (!Number.isInteger(portValue) || portValue < 1 || portValue > 65535) {
			toast.error({ title: 'Invalid port', description: 'Use a port from 1 to 65535.' });
			return;
		}
		if (!Number.isInteger(retentionValue) || retentionValue < 1 || retentionValue > 30) {
			toast.error({ title: 'Invalid retention', description: 'Use 1 to 30 days.' });
			return;
		}

		saving = true;
		const toastId = `monitoring-save-${server.id}`;
		toast.neutral({
			id: toastId,
			title: 'Configuring monitoring',
			description: 'Installing edge agent...',
			icon: { kind: 'spinner' },
			dismissible: false
		});
		try {
			const { error } = await api.POST('/api/servers/{id}/monitoring', {
				params: { path: { id: server.id } },
				body: {
					port: portValue,
					retention_days: retentionValue,
					fqdn: domain
				}
			});
			if (error) {
				toast.error({
					id: toastId,
					title: 'Configuration failed',
					description: (error as components['schemas']['ErrorResponse']).error
				});
				return;
			}
			onClose();
			await onSaved?.();
			toast.success({
				id: toastId,
				title: 'Monitoring ready',
				description: 'Edge agent is now active and collecting metrics.'
			});
		} catch {
			toast.error({
				id: toastId,
				title: 'Configuration failed',
				description: 'Network error. Check server connection and retry.'
			});
		} finally {
			saving = false;
		}
	}
</script>

<Dialog open={serverId !== null} onOpenChange={(open) => !open && onClose()}>
	<DialogContent class="w-[min(92vw,34rem)] max-w-none overflow-hidden">
		<DialogHeader class="border-b border-border px-5 pt-4 pr-12 pb-3">
			<DialogTitle class="flex min-w-0 items-center gap-2 text-sm">
				<span class="flex-none font-medium text-muted-foreground">Monitoring</span>
				<Icon
					src={ChevronRight}
					theme="outline"
					class="h-3.5 w-3.5 flex-none text-muted-foreground"
				/>
				<span class="truncate">{server?.name ?? '…'}</span>
			</DialogTitle>
		</DialogHeader>
		{#if loadError}
			<p class="px-5 py-8 text-center text-sm text-muted-foreground">{loadError}</p>
		{:else if !server}
			<div class="space-y-4 px-5 py-4" aria-busy="true">
				<div class="h-4 w-2/3 animate-pulse rounded bg-muted"></div>
				<div class="h-9 animate-pulse rounded-md bg-muted"></div>
				<div class="h-9 animate-pulse rounded-md bg-muted"></div>
			</div>
		{:else}
			<form
				class="space-y-5 px-5 py-4"
				onsubmit={(event) => {
					event.preventDefault();
					void save();
				}}
			>
				<!-- Two columns of one field each, rather than two full-width rows: both
				     settings are short, and side by side they read as one pass over how
				     the agent runs. Each column carries its own label so the pair stacks
				     intact on a narrow screen instead of unzipping into four rows. -->
				<div class="grid gap-x-5 gap-y-4 sm:grid-cols-[1fr_auto]">
					<div class="flex min-w-0 flex-col gap-2">
						<span class="text-sm font-medium text-foreground">Metrics retention</span>
						<ToggleGroup
							bind:value={retentionDays}
							onValueChange={holdRetention}
							options={retentionOptions}
							class="self-start"
						/>
					</div>

					<div class="flex flex-col gap-2">
						<label for="monitoring-port" class="text-sm font-medium text-foreground">
							Agent port
						</label>
						<div class="group relative w-32">
							<Input
								id="monitoring-port"
								bind:value={port}
								bind:ref={portInput}
								readonly={portLocked}
								onclick={portLocked ? () => void editPort() : undefined}
								inputmode="numeric"
								size="sm"
								class="py-2 font-mono {portLocked ? 'cursor-pointer pr-8' : ''}"
							/>
							{#if portLocked}
								<button
									type="button"
									class="absolute inset-y-0 right-0 grid w-8 cursor-pointer place-items-center rounded-r-md text-muted-foreground transition-colors group-hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
									aria-label="Edit agent port"
									title="Edit agent port"
									onclick={() => void editPort()}
								>
									<Icon src={Pencil} theme="outline" class="h-3.5 w-3.5" />
								</button>
							{/if}
						</div>
					</div>
				</div>

				<div class="flex flex-col gap-3">
					<label class="flex cursor-pointer items-center gap-2 text-sm text-foreground">
						<input
							type="checkbox"
							class="publish-toggle"
							bind:checked={exposePublicly}
							onchange={() => void revealDomain()}
						/>
						Expose metrics for external scrapers (Prometheus)
					</label>
					{#if exposePublicly}
						<FormField label="Public domain">
							<Input
								bind:value={fqdn}
								bind:ref={fqdnInput}
								placeholder="metrics.example.com"
								size="sm"
								autocomplete="off"
								spellcheck="false"
							/>
						</FormField>
					{/if}
				</div>

				<div class="flex justify-end gap-2 border-t border-border pt-4">
					<Button type="button" variant="ghost" size="sm" onclick={onClose}>Cancel</Button>
					<Button type="submit" size="sm" loading={saving}>Save</Button>
				</div>
			</form>
		{/if}
	</DialogContent>
</Dialog>

<style>
	/* Matches the publish toggle in ServiceWorkspace: the same decision, so the same
	   control. Worth extracting into the ui kit once a third surface needs it. */
	.publish-toggle {
		width: 1rem;
		height: 1rem;
		accent-color: var(--primary);
		cursor: pointer;
	}
</style>
