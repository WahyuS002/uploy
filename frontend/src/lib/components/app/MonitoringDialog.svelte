<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import { Button, Collapsible, Input, toast } from '$lib/components/ui';
	import { Dialog, DialogContent, DialogHeader, DialogTitle } from '$lib/components/ui/dialog';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ChevronRight } from '@steeze-ui/heroicons';

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

	let server = $state<ServerResponse | null>(null);
	let loadError = $state('');
	let saving = $state(false);
	let advancedOpen = $state(false);
	let privateAddress = $state('');
	let port = $state('9184');
	let retentionDays = $state('7');
	let fqdn = $state('');
	let readerToken = $state('');

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
		advancedOpen = false;
		// A server being set up for the first time has no recorded private address. The
		// host Uploy already reaches it on is right for the usual single-network box,
		// and wrong in a way the user can see and correct.
		privateAddress = target.monitoring.private_address || target.host;
		port = String(target.monitoring.port || 9184);
		retentionDays = String(target.monitoring.retention_days || 7);
		fqdn = target.monitoring.fqdn ?? '';
		// Generated, not invented. Asking someone to compose 32 random characters before
		// they can see a single metric is the step that stalls this dialog.
		readerToken = isFirstSetup(target) ? generateReaderToken() : '';
	}

	function isFirstSetup(target: ServerResponse) {
		return !target.monitoring.enabled && !target.monitoring.cleanup_at;
	}

	function generateReaderToken() {
		const bytes = crypto.getRandomValues(new Uint8Array(32));
		return [...bytes].map((byte) => byte.toString(16).padStart(2, '0')).join('');
	}

	async function save() {
		if (!server || saving) return;
		const address = privateAddress.trim();
		const portValue = Number(port);
		const retentionValue = Number(retentionDays);
		const token = readerToken.trim();
		if (!address) {
			toast.error({
				title: 'Private address required',
				description: 'Use the address reachable from Uploy.'
			});
			return;
		}
		// Required on first setup, and validated whenever it is being replaced.
		if ((isFirstSetup(server) || token.length > 0) && (token.length < 32 || token.length > 512)) {
			toast.error({
				title: 'Invalid reader token',
				description: 'Use 32 to 512 characters, or select Regenerate.'
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
			title: 'Provisioning monitoring',
			description: 'Installing the edge agent on the server.',
			icon: { kind: 'spinner' },
			dismissible: false
		});
		try {
			const { error } = await api.POST('/api/servers/{id}/monitoring', {
				params: { path: { id: server.id } },
				body: {
					private_address: address,
					port: portValue,
					retention_days: retentionValue,
					fqdn: fqdn.trim(),
					...(token ? { reader_token: token } : {})
				}
			});
			if (error) {
				toast.error({
					id: toastId,
					title: 'Monitoring setup failed',
					description: (error as components['schemas']['ErrorResponse']).error
				});
				return;
			}
			onClose();
			await onSaved?.();
			toast.success({
				id: toastId,
				title: 'Monitoring ready',
				description: 'Uploy now reads live and retained metrics through the edge agent.'
			});
		} catch {
			toast.error({
				id: toastId,
				title: 'Monitoring setup failed',
				description: 'Network error. Check the server connection, then retry.'
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
				class="space-y-4 px-5 py-4"
				onsubmit={(event) => {
					event.preventDefault();
					void save();
				}}
			>
				<p class="text-sm leading-6 text-muted-foreground">
					Stores container metrics locally on this server. Uploy reads the private HTTP API; a
					public FQDN is optional for external scrapers.
				</p>
				<label class="flex flex-col gap-1.5">
					<span class="text-sm font-medium text-foreground">Private address</span>
					<Input bind:value={privateAddress} placeholder="10.0.0.4" autocomplete="off" />
					<p class="text-xs leading-5 text-muted-foreground">
						The address Uploy reaches this server on. Change it if the agent should listen on a
						private interface instead.
					</p>
				</label>
				<div class="flex flex-col gap-1.5">
					<div class="flex items-center justify-between gap-2">
						<label for="monitoring-reader-token" class="text-sm font-medium text-foreground">
							Reader token
						</label>
						<button
							type="button"
							class="cursor-pointer text-xs text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
							onclick={() => (readerToken = generateReaderToken())}
						>
							Regenerate
						</button>
					</div>
					<Input
						id="monitoring-reader-token"
						bind:value={readerToken}
						class="font-mono text-xs"
						minlength={32}
						maxlength={512}
						autocomplete="off"
						spellcheck="false"
						placeholder={isFirstSetup(server) ? '' : 'Leave blank to keep the current token'}
					/>
					<p class="text-xs leading-5 text-muted-foreground">
						Authorizes `/metrics` and the read-only JSON endpoints. Copy it now if an external
						scraper needs it — Uploy does not show it again.
					</p>
				</div>
				<!-- Collapsed by default. The generated token and the server's own address are
				     right for almost every setup, so the dialog opens on nothing to decide. -->
				<Collapsible bind:open={advancedOpen} class="rounded-md border border-border">
					{#snippet trigger()}
						<span
							class="flex w-full cursor-pointer items-center gap-1.5 px-3 py-2 text-sm text-muted-foreground hover:text-foreground"
						>
							<Icon
								src={ChevronRight}
								theme="outline"
								class="h-3.5 w-3.5 transition-transform duration-150 {advancedOpen
									? 'rotate-90'
									: ''}"
							/>
							Advanced
						</span>
					{/snippet}
					<div class="space-y-4 border-t border-border px-3 py-3">
						<div class="grid grid-cols-2 gap-3">
							<label class="flex flex-col gap-1.5">
								<span class="text-sm font-medium text-foreground">Port</span>
								<Input bind:value={port} inputmode="numeric" />
							</label>
							<label class="flex flex-col gap-1.5">
								<span class="text-sm font-medium text-foreground">Retention days</span>
								<Input bind:value={retentionDays} inputmode="numeric" />
							</label>
						</div>
						<label class="flex flex-col gap-1.5">
							<span class="text-sm font-medium text-foreground"
								>Public FQDN <span class="font-normal text-muted-foreground">(optional)</span></span
							>
							<Input bind:value={fqdn} placeholder="metrics.example.com" autocomplete="off" />
						</label>
					</div>
				</Collapsible>
				<div class="flex justify-end gap-2 border-t border-border pt-4">
					<Button type="button" variant="ghost" size="sm" onclick={onClose}>Cancel</Button>
					<Button type="submit" size="sm" loading={saving}>Save monitoring</Button>
				</div>
			</form>
		{/if}
	</DialogContent>
</Dialog>
