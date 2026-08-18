<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import type { PageData } from './$types';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import { Button, Input, Textarea, toast } from '$lib/components/ui';

	type Channel = components['schemas']['NotificationChannelResponse'];
	type Rule = components['schemas']['AlertRuleResponse'];
	type Event = components['schemas']['AlertEventResponse'];

	let { data }: { data: PageData } = $props();
	let channels = $derived(data.channels as Channel[]);
	let rules = $derived(data.rules as Rule[]);
	let history = $derived(data.history as Event[]);

	let channelName = $state('');
	let channelType = $state<components['schemas']['NotificationChannelType']>('discord');
	let channelConfig = $state('{\n  "url": ""\n}');
	let ruleName = $state('');
	let condition = $state<components['schemas']['AlertCondition']>('cpu_high');
	let threshold = $state('85');
	let durationSeconds = $state('600');
	let scopeType = $state<components['schemas']['AlertScopeType']>('service');
	let targetId = $state('');
	let selectedChannels = $state<string[]>([]);
	let saving = $state(false);
	let testingId = $state<string | null>(null);

	let needsThreshold = $derived(condition === 'cpu_high' || condition === 'memory_high' || condition === 'disk_low');
	let targetLabel = $derived(scopeType === 'service' ? 'Service ID' : 'Server ID');

	async function createChannel() {
		if (saving) return;
		let config: Record<string, unknown>;
		try {
			config = JSON.parse(channelConfig);
		} catch {
			toast.error({ title: 'Invalid JSON', description: 'Channel configuration must be valid JSON.' });
			return;
		}
		saving = true;
		try {
			const { error } = await api.POST('/api/notification-channels', {
				body: { name: channelName.trim(), type: channelType, config }
			});
			if (error) {
				toast.error({ title: 'Channel not saved', description: error.error });
				return;
			}
			channelName = '';
			toast.success({ title: 'Channel added', description: 'You can now attach it to alert rules.' });
			await invalidateAll();
		} catch {
			toast.error({ title: 'Channel not saved', description: 'Network error.' });
		} finally {
			saving = false;
		}
	}

	async function testChannel(channel: Channel) {
		if (testingId) return;
		testingId = channel.id;
		try {
			const { error } = await api.POST('/api/notification-channels/{id}/test', { params: { path: { id: channel.id } } });
			if (error) {
				toast.error({ title: 'Test failed', description: error.error });
				return;
			}
			toast.success({ title: 'Test sent', description: `Check ${channel.name}.` });
		} catch {
			toast.error({ title: 'Test failed', description: 'Network error.' });
		} finally {
			testingId = null;
		}
	}

	async function deleteChannel(channel: Channel) {
		if (!window.confirm(`Delete ${channel.name}? Rules using it will stop notifying.`)) return;
		const { error } = await api.DELETE('/api/notification-channels/{id}', { params: { path: { id: channel.id } } });
		if (error) {
			toast.error({ title: 'Channel not deleted', description: error.error });
			return;
		}
		await invalidateAll();
	}

	async function createRule() {
		if (saving) return;
		const duration = Number(durationSeconds);
		const parsedThreshold = needsThreshold ? Number(threshold) : 0;
		if (!Number.isInteger(duration) || duration < 60) {
			toast.error({ title: 'Invalid duration', description: 'Use at least 60 seconds.' });
			return;
		}
		if (!targetId.trim()) {
			toast.error({ title: 'Target required', description: `Enter a ${scopeType} ID.` });
			return;
		}
		if (!selectedChannels.length) {
			toast.error({ title: 'Channel required', description: 'Select at least one channel.' });
			return;
		}
		saving = true;
		try {
			const { error } = await api.POST('/api/alert-rules', {
				body: {
					name: ruleName.trim(), condition, threshold: parsedThreshold, duration_seconds: duration,
					scope_type: scopeType, ...(scopeType === 'service' ? { service_id: targetId.trim() } : { server_id: targetId.trim() }),
					channel_ids: selectedChannels
				}
			});
			if (error) {
				toast.error({ title: 'Rule not saved', description: error.error });
				return;
			}
			ruleName = '';
			targetId = '';
			selectedChannels = [];
			toast.success({ title: 'Rule added', description: 'Evaluation starts in the background.' });
			await invalidateAll();
		} catch {
			toast.error({ title: 'Rule not saved', description: 'Network error.' });
		} finally {
			saving = false;
		}
	}

	function toggleChannel(id: string) {
		selectedChannels = selectedChannels.includes(id)
			? selectedChannels.filter((value) => value !== id)
			: [...selectedChannels, id];
	}

	function formatDate(value: string) {
		return new Intl.DateTimeFormat('en-GB', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value));
	}

	function duration(event: Event) {
		if (event.duration_seconds == null) return 'Active';
		const minutes = Math.floor(event.duration_seconds / 60);
		return minutes ? `${minutes}m` : `${event.duration_seconds}s`;
	}
</script>

<svelte:head><title>Alerts · Uploy</title></svelte:head>

<section class="flex flex-1 flex-col gap-8">
	<div>
		<h1 class="text-xl font-semibold tracking-tight text-foreground">Alerts</h1>
		<p class="mt-1 max-w-2xl text-sm text-muted-foreground">One message when a problem starts, one when it recovers. Configure destinations separately from the rules that use them.</p>
	</div>

	<div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
		<div class="space-y-4">
			<div class="border-b border-border pb-3"><h2 class="font-medium text-foreground">Notification channels</h2><p class="mt-1 text-xs text-muted-foreground">Discord, Slack, Telegram, email, or a generic webhook.</p></div>
			<div class="space-y-3">
				<Input bind:value={channelName} placeholder="Team Discord" aria-label="Channel name" />
				<select bind:value={channelType} class="h-9 w-full rounded-md border border-input bg-background px-3 text-sm text-foreground outline-none focus:ring-2 focus:ring-ring/40" aria-label="Channel type">
					<option value="discord">Discord</option><option value="slack">Slack</option><option value="telegram">Telegram</option><option value="email">Email</option><option value="webhook">Webhook</option>
				</select>
				<Textarea bind:value={channelConfig} rows={5} aria-label="Channel configuration" />
				<div class="flex justify-end"><Button variant="primary" size="sm" onclick={createChannel} disabled={saving}>Add channel</Button></div>
			</div>
			{#if channels.length}
				<div class="divide-y divide-border border-y border-border">
					{#each channels as channel (channel.id)}
						<div class="flex items-center justify-between gap-3 py-3">
							<div class="min-w-0"><p class="truncate text-sm font-medium text-foreground">{channel.name}</p><p class="text-xs text-muted-foreground">{channel.type} · {channel.enabled ? 'enabled' : 'disabled'}</p></div>
							<div class="flex shrink-0 gap-2"><Button variant="secondary" size="sm" onclick={() => testChannel(channel)} disabled={testingId === channel.id}>{testingId === channel.id ? 'Sending…' : 'Test'}</Button><Button variant="ghost" size="sm" onclick={() => deleteChannel(channel)}>Delete</Button></div>
						</div>
					{/each}
				</div>
			{:else}<p class="border-y border-border py-5 text-sm text-muted-foreground">No channels yet. Add one above, then send a test before attaching it to a rule.</p>{/if}
		</div>

		<div class="space-y-4">
			<div class="border-b border-border pb-3"><h2 class="font-medium text-foreground">Alert rules</h2><p class="mt-1 text-xs text-muted-foreground">A condition must stay true for the duration before it can notify.</p></div>
			<div class="grid gap-3 sm:grid-cols-2">
				<Input bind:value={ruleName} placeholder="Production CPU" aria-label="Rule name" />
				<select bind:value={condition} class="h-9 rounded-md border border-input bg-background px-3 text-sm text-foreground outline-none focus:ring-2 focus:ring-ring/40" aria-label="Condition"><option value="cpu_high">CPU high</option><option value="memory_high">Memory high</option><option value="disk_low">Disk usage high</option><option value="service_down">Service down</option><option value="server_unreachable">Server unreachable</option></select>
				{#if needsThreshold}<Input bind:value={threshold} type="number" min="1" max="100" placeholder="Threshold %" aria-label="Threshold" />{/if}
				<Input bind:value={durationSeconds} type="number" min="60" placeholder="Wait seconds" aria-label="Wait duration in seconds" />
				<select bind:value={scopeType} class="h-9 rounded-md border border-input bg-background px-3 text-sm text-foreground outline-none focus:ring-2 focus:ring-ring/40" aria-label="Scope"><option value="service">Service</option><option value="server">Server</option></select>
				<Input bind:value={targetId} placeholder={targetLabel} aria-label={targetLabel} />
			</div>
			<div class="space-y-2"><p class="text-xs font-medium text-muted-foreground">Notify through</p>{#if channels.length}<div class="grid gap-2 sm:grid-cols-2">{#each channels as channel (channel.id)}<label class="flex cursor-pointer items-center gap-2 text-sm text-foreground"><input type="checkbox" checked={selectedChannels.includes(channel.id)} onchange={() => toggleChannel(channel.id)} />{channel.name}</label>{/each}</div>{:else}<p class="text-sm text-muted-foreground">Create a channel first.</p>{/if}</div>
			<div class="flex justify-end"><Button variant="primary" size="sm" onclick={createRule} disabled={saving || !channels.length}>Add rule</Button></div>
			{#if rules.length}<div class="divide-y divide-border border-y border-border">{#each rules as rule (rule.id)}<div class="py-3"><div class="flex items-start justify-between gap-3"><div><p class="text-sm font-medium text-foreground">{rule.name}</p><p class="mt-1 text-xs text-muted-foreground">{rule.condition} · {rule.scope_type} · {rule.duration_seconds}s wait</p></div><span class="text-xs {rule.enabled ? 'text-emerald-700' : 'text-muted-foreground'}">{rule.enabled ? 'Enabled' : 'Disabled'}</span></div></div>{/each}</div>{:else}<p class="border-y border-border py-5 text-sm text-muted-foreground">No rules yet. Start with CPU or memory for a service you care about.</p>{/if}
		</div>
	</div>

	<div class="space-y-4">
		<div class="border-b border-border pb-3"><h2 class="font-medium text-foreground">Incident history</h2><p class="mt-1 text-xs text-muted-foreground">Every start and recovery is kept as an incident record.</p></div>
		{#if history.length}<div class="overflow-x-auto"><table class="w-full min-w-[42rem] text-left text-sm"><thead><tr class="border-b border-border text-xs text-muted-foreground"><th class="pb-2 font-medium">Target</th><th class="pb-2 font-medium">Status</th><th class="pb-2 font-medium">Started</th><th class="pb-2 font-medium">Duration</th><th class="pb-2 text-right font-medium">Value</th></tr></thead><tbody>{#each history as event (event.id)}<tr class="border-b border-border/70"><td class="py-3"><p class="font-medium text-foreground">{event.target_name}</p><p class="text-xs text-muted-foreground">{event.rule_id}</p></td><td class="py-3"><span class={event.status === 'firing' ? 'text-amber-700' : 'text-emerald-700'}>{event.status}</span></td><td class="py-3 text-muted-foreground">{formatDate(event.started_at)}</td><td class="py-3 text-muted-foreground">{duration(event)}</td><td class="py-3 text-right tabular-nums text-foreground">{event.trigger_value.toFixed(1)}</td></tr>{/each}</tbody></table></div>{:else}<p class="border-y border-border py-5 text-sm text-muted-foreground">No incidents recorded. That is good news — the evaluator is ready when a condition stays true.</p>{/if}
	</div>
</section>
