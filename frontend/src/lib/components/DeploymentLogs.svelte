<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
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

	// Map structured phase identifiers from backend to human-readable labels
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
		// Track the last stderr line that carries the real error detail.
		// Skip synthetic wrappers so the banner shows the actual error.
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

	let bannerColor: string = $derived.by(() => {
		if (bannerStatus === 'success') return '#10b981';
		if (bannerStatus === 'error') return '#ef4444';
		return '#3b82f6';
	});

	let bannerBgColor: string = $derived.by(() => {
		if (bannerStatus === 'success') return 'rgba(16, 185, 129, 0.1)';
		if (bannerStatus === 'error') return 'rgba(239, 68, 68, 0.1)';
		return 'rgba(59, 130, 246, 0.1)';
	});

	let statusIcon: string = $derived.by(() => {
		if (bannerStatus === 'success') return '\u2714';
		if (bannerStatus === 'error') return '\u2716';
		return '\u25CF';
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
		style="
			border: 1px solid {bannerColor};
			background: {bannerBgColor};
			border-radius: 8px;
			padding: 0.75rem 1rem;
			margin-bottom: 0.75rem;
			font-family: system-ui, -apple-system, sans-serif;
		"
	>
		<div style="display: flex; justify-content: space-between; align-items: center;">
			<div style="display: flex; align-items: center; gap: 0.5rem;">
				<span
					style="
						color: {bannerColor};
						font-size: 0.75rem;
						{bannerStatus === 'active' ? 'animation: pulse 2s infinite;' : ''}
					">{statusIcon}</span
				>
				<span style="font-weight: 600; color: {bannerColor}; font-size: 0.9rem;"
					>{currentPhase}</span
				>
			</div>
			<span style="font-size: 0.8rem; color: #888; font-variant-numeric: tabular-nums;"
				>{formatElapsed(elapsedSeconds)}</span
			>
		</div>
		{#if currentSubtext && bannerStatus !== 'success'}
			<div style="margin-top: 0.25rem; font-size: 0.8rem; color: #999;">
				{currentSubtext}
			</div>
		{/if}
		{#if streamError}
			<div style="margin-top: 0.25rem; font-size: 0.8rem; color: #ef4444;">
				{streamError}
			</div>
		{/if}
	</div>

	<!-- Log Panel -->
	<div
		style="
			background: #1a1a1a;
			color: #fff;
			padding: 1rem;
			font-family: monospace;
			border-radius: 8px;
			max-height: 400px;
			overflow-y: auto;
		"
	>
		{#each logs as log (log.order)}
			<p style="margin: 0; color: {log.type === 'stderr' ? '#ff6b6b' : '#fff'}">
				{log.output}
			</p>
		{/each}
	</div>
</div>

<style>
	@keyframes pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.4;
		}
	}
</style>
