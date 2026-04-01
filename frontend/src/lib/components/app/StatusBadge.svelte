<script lang="ts">
	import Badge, { type BadgeTone } from '$lib/components/ui/Badge.svelte';

	type Props = {
		status: string;
		class?: string;
	};

	let { status, class: className }: Props = $props();

	const statusToneMap: Record<string, BadgeTone> = {
		ready: 'success',
		success: 'success',
		active: 'success',
		pending: 'warning',
		in_progress: 'warning',
		not_configured: 'warning',
		error: 'danger',
		failed: 'danger',
		port_conflict: 'danger',
		degraded: 'danger'
	};

	let tone = $derived(statusToneMap[status] ?? 'neutral');
	let label = $derived(status.replace(/_/g, ' '));
</script>

<Badge {tone} class={className}>
	{label}
</Badge>
