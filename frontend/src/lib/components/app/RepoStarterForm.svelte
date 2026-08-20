<script lang="ts">
	import { onMount, type Snippet } from 'svelte';
	import { ArrowLeft, Check, GitFork as Github, LoaderCircle } from 'lucide-svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';

	type Analysis = {
		owner: string;
		name: string;
		branch: string;
		provider: string;
		runtime_versions: Record<string, string>;
		start_command?: string;
		suggested_name: string;
		suggested_port: number;
	};

	type Props = {
		/** Rows shown under the branch row — where the deploy target is spelled out. */
		details?: Snippet;
		analysis?: Analysis | null;
		submitting?: boolean;
		analyzing?: boolean;
		error?: string;
		onBack: () => void;
		onChangeRepository: () => void;
		onAnalyze: (repoUrl: string, branch: string) => void;
		onCreate: (values: {
			name: string;
			containerName: string;
			containerPort: number;
			branch: string;
		}) => void;
	};

	let {
		details,
		analysis = null,
		submitting = false,
		analyzing = false,
		error = '',
		onBack,
		onChangeRepository,
		onAnalyze,
		onCreate
	}: Props = $props();

	let repoUrl = $state('');
	let branch = $state('main');
	let name = $state('');
	let containerName = $state('');
	let containerPort = $state(3000);
	let containerPortTouched = $state(false);
	let containerNameTouched = $state(false);
	let localError = $state('');
	let repoInput = $state<HTMLInputElement | null>(null);

	let step = $derived(analysis ? 'confirm' : 'repo');
	let repoReady = $derived(repoUrl.trim() !== '' && branch.trim() !== '');
	let formReady = $derived(
		name.trim() !== '' && containerName.trim() !== '' && Number.isFinite(containerPort)
	);

	function safeContainerName(value: string): string {
		const cleaned = value
			.toLowerCase()
			.replace(/[^a-z0-9_.-]+/g, '-')
			.replace(/^[^a-z0-9]+/, '')
			.replace(/[^a-z0-9]+$/, '');
		return cleaned || 'service';
	}

	$effect(() => {
		if (!analysis) return;
		if (name === '') name = analysis.suggested_name;
		if (!containerNameTouched) containerName = safeContainerName(name);
		if (!containerPortTouched) containerPort = analysis.suggested_port;
		if (branch === 'main' && analysis.branch) branch = analysis.branch;
	});

	function submitAnalysis(event: SubmitEvent) {
		event.preventDefault();
		localError = '';
		if (!repoUrl.trim()) {
			localError = 'Repository URL is required';
			return;
		}
		if (!branch.trim()) {
			localError = 'Branch is required';
			return;
		}
		onAnalyze(repoUrl.trim(), branch.trim());
	}

	function submitCreate(event: SubmitEvent) {
		event.preventDefault();
		if (submitting) return;
		localError = '';
		if (!name.trim() || !containerName.trim()) {
			localError = 'Service name and container name are required';
			return;
		}
		if (!Number.isInteger(containerPort) || containerPort < 1 || containerPort > 65535) {
			localError = 'Port must be between 1 and 65535';
			return;
		}
		onCreate({
			name: name.trim(),
			containerName: containerName.trim(),
			containerPort,
			branch: branch.trim()
		});
	}

	function goBack() {
		if (step === 'confirm') {
			name = '';
			containerName = '';
			containerNameTouched = false;
			containerPortTouched = false;
			onChangeRepository();
			return;
		}
		onBack();
	}

	onMount(() => repoInput?.focus());
</script>

{#if step === 'repo'}
	<form
		onsubmit={submitAnalysis}
		class="card overflow-hidden rounded-xl border border-border bg-card text-card-foreground"
	>
		<div class="border-b border-border/70 px-4 py-3">
			<div class="flex items-center gap-2.5">
				<span class="grid h-7 w-7 place-content-center rounded-md bg-muted text-foreground">
					<Github class="h-4 w-4" strokeWidth={1.75} />
				</span>
				<div>
					<h2 class="text-sm font-medium text-foreground">Import a GitHub repository</h2>
					<p class="mt-0.5 text-xs text-muted-foreground">Public repositories only for now.</p>
				</div>
			</div>
		</div>

		<div class="flex flex-col gap-3 p-4">
			<label class="flex flex-col gap-1.5 text-xs text-muted-foreground">
				<span>Repository URL</span>
				<Input
					bind:value={repoUrl}
					bind:ref={repoInput}
					placeholder="github.com/owner/repository"
					autocomplete="url"
					autocapitalize="off"
					spellcheck={false}
					required
				/>
			</label>
			<label class="flex flex-col gap-1.5 text-xs text-muted-foreground">
				<span>Branch</span>
				<Input bind:value={branch} placeholder="main" autocomplete="off" required />
			</label>

			{#if localError || error}
				<Alert tone="danger">{localError || error}</Alert>
				<p class="-mt-1 text-xs text-muted-foreground">
					See the <a
						class="underline underline-offset-2 hover:text-foreground"
						href="https://railpack.com"
						target="_blank"
						rel="noreferrer">Railpack documentation</a
					> for supported project files.
				</p>
			{/if}

			<div class="flex items-center justify-between gap-3 pt-1">
				<Button type="button" variant="ghost" size="sm" onclick={onBack}>Back</Button>
				<Button type="submit" size="sm" loading={analyzing} disabled={!repoReady || analyzing}>
					{analyzing ? 'Analyzing…' : 'Analyze repository'}
				</Button>
			</div>
		</div>
	</form>
{:else}
	<form
		onsubmit={submitCreate}
		class="card overflow-hidden rounded-xl border border-border bg-card text-card-foreground"
	>
		<div class="border-b border-border/70 px-4 py-3">
			<div class="flex items-start justify-between gap-3">
				<div class="flex min-w-0 items-center gap-2.5">
					<span
						class="grid h-7 w-7 flex-none place-content-center rounded-md bg-muted text-foreground"
					>
						<Check class="h-4 w-4" strokeWidth={1.9} />
					</span>
					<div class="min-w-0">
						<h2 class="text-sm font-medium text-foreground">Confirm repository service</h2>
						<p class="mt-0.5 truncate font-mono text-xs text-muted-foreground">
							{analysis?.owner}/{analysis?.name} · {analysis?.branch}
						</p>
					</div>
				</div>
				<button
					type="button"
					onclick={goBack}
					class="grid h-7 w-7 flex-none place-content-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
					aria-label="Change repository"
				>
					<ArrowLeft class="h-3.5 w-3.5" strokeWidth={1.75} />
				</button>
			</div>
		</div>

		<div class="flex flex-col gap-4 p-4">
			<div class="grid grid-cols-2 gap-2 border-b border-border/70 pb-3 sm:grid-cols-3">
				<div class="min-w-0">
					<p class="text-[11px] text-muted-foreground">Detected runtime</p>
					<p class="mt-1 truncate text-xs font-medium text-foreground">
						{analysis?.provider ?? 'Unknown'}
					</p>
				</div>
				<div class="min-w-0">
					<p class="text-[11px] text-muted-foreground">Version</p>
					<p class="mt-1 truncate font-mono text-xs text-foreground">
						{Object.values(analysis?.runtime_versions ?? {})[0] ?? 'Not specified'}
					</p>
				</div>
				<div class="col-span-2 min-w-0 sm:col-span-1">
					<p class="text-[11px] text-muted-foreground">Start command</p>
					<p class="mt-1 truncate font-mono text-xs text-foreground">
						{analysis?.start_command ?? 'Not specified'}
					</p>
				</div>
			</div>

			<div class="flex flex-col gap-3">
				<label class="flex flex-col gap-1.5 text-xs text-muted-foreground">
					<span>Service name</span>
					<Input bind:value={name} required />
				</label>
				<label class="flex flex-col gap-1.5 text-xs text-muted-foreground">
					<span>Container name</span>
					<Input
						bind:value={containerName}
						oninput={() => (containerNameTouched = true)}
						required
					/>
				</label>
				<div class="grid grid-cols-2 gap-3">
					<label class="flex flex-col gap-1.5 text-xs text-muted-foreground">
						<span>Container port</span>
						<Input
							type="number"
							bind:value={containerPort}
							oninput={() => (containerPortTouched = true)}
							min={1}
							max={65535}
							required
						/>
					</label>
					<label class="flex flex-col gap-1.5 text-xs text-muted-foreground">
						<span>Branch</span>
						<Input bind:value={branch} required />
					</label>
				</div>
			</div>

			{#if details}
				<div class="flex flex-col gap-2 border-t border-border/70 pt-3">
					{@render details()}
				</div>
			{/if}

			{#if localError || error}
				<Alert tone="danger">{localError || error}</Alert>
			{/if}

			<div class="flex items-center justify-between gap-3 pt-1">
				<Button type="button" variant="ghost" size="sm" onclick={goBack}>Change repository</Button>
				<Button type="submit" size="sm" loading={submitting} disabled={!formReady || submitting}>
					{submitting ? 'Creating…' : 'Create service'}
				</Button>
			</div>
		</div>
	</form>
{/if}

{#if analyzing}
	<div class="mt-3 flex items-center gap-2 text-xs text-muted-foreground" role="status">
		<LoaderCircle class="h-3.5 w-3.5 animate-spin" />
		<span>Railpack is reading the repository…</span>
	</div>
{/if}

<style>
	.card {
		box-shadow: var(--shadow-panel);
	}
</style>
