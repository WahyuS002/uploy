<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { cn } from '$lib/components/ui/cn.js';
	import type { components } from '$lib/api/v1';

	type LogEntry = components['schemas']['LogEntry'];

	interface Props {
		deploymentId: string;
		onDone?: (status: string) => void;
	}

	let { deploymentId, onDone }: Props = $props();

	let logs: LogEntry[] = $state([]);
	let status: string = $state('in_progress');
	let streamError: string = $state('');
	let eventSource: EventSource | null = null;

	const phaseLabels: Record<string, string> = {
		connect: 'Connecting to Server',
		pull_image: 'Pulling Image',
		proxy_setup: 'Setting Up Reverse Proxy',
		stop_container: 'Stopping Existing Container',
		start_container: 'Starting Application',
		tls_cert: 'Waiting for TLS Certificates',
		complete: 'Deployment Complete',
		failed: 'Deployment Failed'
	};

	let currentPhase: string = $state('Starting...');
	let currentSubtext: string = $state('');
	let lastErrorReason: string = $state('');
	let phaseStartTime: number = $state(Date.now());
	let elapsedSeconds: number = $state(0);
	let timerInterval: ReturnType<typeof setInterval> | null = null;

	function updatePhaseFromLog(log: LogEntry) {
		if (
			log.type === 'stderr' &&
			log.phase !== 'failed' &&
			!log.output.startsWith('command failed: ')
		) {
			lastErrorReason = log.output;
		}

		if (!log.phase) return;

		const label = phaseLabels[log.phase];
		if (!label) return;

		const logTime = new Date(log.created_at).getTime();
		if (label !== currentPhase) {
			currentPhase = label;
			phaseStartTime = logTime;
			elapsedSeconds = Math.max(0, Math.floor((Date.now() - logTime) / 1000));
		}
		currentSubtext = log.output;
	}

	function startTimer() {
		if (timerInterval) return;
		timerInterval = setInterval(() => {
			elapsedSeconds = Math.max(0, Math.floor((Date.now() - phaseStartTime) / 1000));
		}, 1000);
	}

	function stopTimer() {
		if (timerInterval) {
			clearInterval(timerInterval);
			timerInterval = null;
		}
	}

	function formatElapsed(seconds: number): string {
		if (seconds < 60) return `${seconds}s`;
		const m = Math.floor(seconds / 60);
		const s = seconds % 60;
		return `${m}m ${s}s`;
	}

	let bannerStatus: 'active' | 'success' | 'error' = $derived.by(() => {
		if (status === 'success') return 'success';
		if (status === 'failed') return 'error';
		return 'active';
	});

	onMount(() => {
		startTimer();

		eventSource = new EventSource(`/api/deployments/${deploymentId}/logs`);

		eventSource.onmessage = (e) => {
			const log: LogEntry = JSON.parse(e.data);
			logs = [...logs, log];
			updatePhaseFromLog(log);
		};

		eventSource.addEventListener('done', (e) => {
			status = (e as MessageEvent).data;
			if (status === 'success') {
				currentPhase = 'Deployment Complete';
				currentSubtext = 'deployment success';
			} else if (status === 'failed') {
				currentPhase = 'Deployment Failed';
				if (lastErrorReason) {
					currentSubtext = lastErrorReason;
				}
			}
			stopTimer();
			onDone?.(status);
			eventSource?.close();
		});

		eventSource.addEventListener('stream-error', (e) => {
			const data = JSON.parse((e as MessageEvent).data);
			streamError = data.message;
			status = 'failed';
			currentPhase = 'Deployment Failed';
			if (lastErrorReason) {
				currentSubtext = lastErrorReason;
			}
			stopTimer();
			onDone?.('failed');
			eventSource?.close();
		});
	});

	onDestroy(() => {
		stopTimer();
		eventSource?.close();
	});
</script>

<div>
	<!-- Phase Banner -->
	<div
		class={cn(
			'mb-3 rounded-lg border p-3',
			bannerStatus === 'success' && 'border-success/40 bg-success/10',
			bannerStatus === 'error' && 'border-destructive/40 bg-destructive/10',
			bannerStatus === 'active' && 'border-info/40 bg-info/10'
		)}
	>
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-2">
				<span
					class={cn(
						'text-xs',
						bannerStatus === 'success' && 'text-success',
						bannerStatus === 'error' && 'text-destructive',
						bannerStatus === 'active' && 'animate-pulse text-info'
					)}
				>
					{#if bannerStatus === 'success'}&#10004;{:else if bannerStatus === 'error'}&#10006;{:else}&#9679;{/if}
				</span>
				<span
					class={cn(
						'text-sm font-semibold',
						bannerStatus === 'success' && 'text-success',
						bannerStatus === 'error' && 'text-destructive',
						bannerStatus === 'active' && 'text-info'
					)}
				>
					{currentPhase}
				</span>
			</div>
			<span class="text-sm text-muted-foreground tabular-nums">
				{formatElapsed(elapsedSeconds)}
			</span>
		</div>
		{#if currentSubtext && bannerStatus !== 'success'}
			<div class="mt-1 text-sm text-muted-foreground">{currentSubtext}</div>
		{/if}
		{#if streamError}
			<div class="mt-1 text-sm text-destructive">{streamError}</div>
		{/if}
	</div>

	<!-- Log Panel -->
	<div class="max-h-100 overflow-y-auto rounded-lg bg-[#1a1a1a] p-4 font-mono text-white">
		{#each logs as log (log.order)}
			<p class={cn('m-0', log.type === 'stderr' ? 'text-[#ff6b6b]' : 'text-white')}>
				{log.output}
			</p>
		{/each}
	</div>
</div>
