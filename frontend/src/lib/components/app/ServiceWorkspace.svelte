<script lang="ts">
	import { untrack } from 'svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import DeploymentLogs from '$lib/components/DeploymentLogs.svelte';
	import LogStream from '$lib/components/app/LogStream.svelte';
	import DnsRecordDialog from '$lib/components/app/DnsRecordDialog.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import StatusBadge from '$lib/components/app/StatusBadge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import DataRow, { dataRowVariants } from '$lib/components/ui/DataRow.svelte';
	import IconButton from '$lib/components/ui/IconButton.svelte';
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogFooter,
		DialogTitle
	} from '$lib/components/ui/dialog';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ExclamationTriangle, GlobeAlt, Server, XMark } from '@steeze-ui/heroicons';
	import { cn } from '$lib/components/ui/cn.js';
	import { countdownTo } from '$lib/countdown.svelte';
	import { formatDateTime, formatRelativeTime } from '$lib/format-date';

	type ServiceResponse = components['schemas']['ServiceResponse'];
	type ServerResponse = components['schemas']['ServerResponse'];
	type ServiceDomainResponse = components['schemas']['ServiceDomainResponse'];
	type ServiceEnvResponse = components['schemas']['ServiceEnvResponse'];
	type DeploymentResponse = components['schemas']['DeploymentResponse'];

	type Tab = 'deployments' | 'logs' | 'domains' | 'env' | 'settings';

	type Props = {
		service: ServiceResponse;
		canEdit: boolean;
		/** Deleting a service is owner-only server-side, so the action only shows for owners. */
		isOwner?: boolean;
		showEnvVars?: boolean;
		/**
		 * A deployment started outside this panel — the canvas "pending changes"
		 * bar deploys straight from the page and this is how its logs find their
		 * way here. Without it a bar deploy runs invisibly.
		 */
		externalDeploymentId?: string | null;
		onDeleted?: (id: string) => void;
		/** The saved service, so the caller can refresh whatever it renders from. */
		onUpdated?: (service: ServiceResponse) => void;
		class?: string;
		/**
		 * The past deployment whose logs are open, or null. Bound rather than
		 * rendered here: its panel stacks *over* the surface this workspace lives
		 * in, and a child cannot draw outside its own parent's edge.
		 */
		openDeployment?: DeploymentResponse | null;
	};

	let {
		service,
		canEdit,
		isOwner = false,
		showEnvVars = true,
		externalDeploymentId = null,
		onDeleted,
		onUpdated,
		class: className,
		openDeployment = $bindable<DeploymentResponse | null>(null)
	}: Props = $props();

	let svcId = $derived(service.id);

	// Deployments lands first, so it is what a freshly selected service opens on.
	let activeTab = $state<Tab>('deployments');

	let domains = $state<ServiceDomainResponse[]>([]);
	let envs = $state<ServiceEnvResponse[]>([]);
	let envsLoaded = $state(false);
	// Derived rather than synced with an effect: the reset below and an incoming
	// external id would otherwise race on service change, and which one won would
	// come down to declaration order. A deploy started from this panel is the more
	// specific fact, so it wins; otherwise the page's id shows through.
	let localDeploymentId = $state<string | null>(null);
	let deploymentId = $derived(localDeploymentId ?? externalDeploymentId);
	let deploying = $state(false);
	let deployError = $state('');
	let deployments = $state<DeploymentResponse[]>([]);
	// The API returns them newest first, so the head is the current one and the
	// tail is history — the same split the panel shows.
	let latestDeployment = $derived(deployments[0] ?? null);
	let previousDeployments = $derived(deployments.slice(1));

	// The service only carries a server_id and there is no GET /api/servers/{id},
	// so the list is the only way to put a name on it. Fetched once per mount, not
	// per service: it does not change when the selection does.
	let servers = $state<ServerResponse[]>([]);
	let server = $derived(servers.find((s) => s.id === service.server_id) ?? null);

	// The address the Deployments tab leads with. Primary if one is marked, else
	// the first attached — either way it is the one a reader would try.
	let primaryDomain = $derived(domains.find((d) => d.is_primary) ?? domains[0] ?? null);

	// What the form cannot change. Container name and server are deliberately not
	// editable: the container is found by name on one server, so changing either
	// would leave the running one behind with nothing in Uploy pointing at it —
	// the same orphan that deleting a service used to create.
	// Image and the ports are only here for readers who cannot edit: for everyone
	// else the form below is the live version of the same two facts, and printing
	// them twice makes the card look like a second, stale copy.
	let fixedMetadata = $derived([
		...(canEdit
			? []
			: [
					{ label: 'Image', value: service.image, mono: true },
					{
						label: 'Port',
						value:
							service.host_port != null
								? `${service.host_port} → ${service.container_port}`
								: `${service.container_port} (internal only)`,
						mono: true
					}
				]),
		{ label: 'Container', value: service.container_name, mono: true },
		{ label: 'Server', value: server ? `${server.name} (${server.host})` : '—', mono: false }
	]);

	let editName = $state('');
	let editImage = $state('');
	let editContainerPort = $state(80);
	let editHostPort = $state(8080);
	let editExposed = $state(true);
	let saving = $state(false);
	let saveError = $state('');
	let savedAt = $state(0);

	function resetEditForm(svc: ServiceResponse) {
		editName = svc.name;
		editImage = svc.image;
		editContainerPort = svc.container_port;
		// No host port means the service is internal only. Seed the field with a
		// plausible number anyway, so turning publishing on has something to offer.
		editExposed = svc.host_port != null;
		editHostPort = svc.host_port ?? svc.container_port;
		saveError = '';
	}

	let edited = $derived(
		editName !== service.name ||
			editImage !== service.image ||
			editContainerPort !== service.container_port ||
			editExposed !== (service.host_port != null) ||
			(editExposed && editHostPort !== service.host_port)
	);

	async function saveService() {
		if (saving) return;
		saveError = '';

		if (!editName.trim()) {
			saveError = 'Name is required';
			return;
		}
		if (!editImage.trim()) {
			saveError = 'Image is required';
			return;
		}

		saving = true;
		try {
			const { data, error: err } = await api.PUT('/api/services/{id}', {
				params: { path: { id: svcId } },
				body: {
					name: editName.trim(),
					image: editImage.trim(),
					// Unchanged, but the API takes the whole resource.
					container_name: service.container_name,
					container_port: editContainerPort,
					host_port: editExposed ? editHostPort : undefined,
					server_id: service.server_id
				}
			});
			if (err || !data) {
				saveError = (err as { error: string } | undefined)?.error ?? 'Failed to save service';
				return;
			}
			onUpdated?.(data);
			// Re-seed from what the server stored, not from what was typed, so a
			// value it normalised does not leave the form looking unsaved.
			resetEditForm(data);
			savedAt = Date.now();
		} catch {
			saveError = 'Network error';
		} finally {
			saving = false;
		}
	}

	let deleteOpen = $state(false);
	let deleting = $state(false);
	let deleteError = $state('');

	let domainInput = $state('');
	let domainError = $state('');
	let domainAdding = $state(false);
	let needsRedeploy = $state(false);
	// The domain whose DNS record is being explained, or null. Opened by adding
	// one and reopenable from any row that is still waiting — a dialog you can
	// only ever see once is a dead end for the person who dismissed it too fast.
	//
	// Held by id and looked back up, so the dialog's status mark tracks the list
	// rather than freezing on whatever the row said when it was clicked.
	let dnsDomainId = $state<string | null>(null);
	let dnsDomain = $derived(domains.find((d) => d.id === dnsDomainId) ?? null);

	/**
	 * How each domain state shows up in the list. A globe for one that answers, a
	 * triangle for one that does not yet — the mark carries the state so the row
	 * does not need a badge spelling it out on the far side as well.
	 *
	 * `ready` means the TLS reconciler found a certificate for this hostname, and
	 * there is no certificate without DNS that resolves here — so it is the
	 * strongest thing the panel can honestly say.
	 */
	const domainMarks = {
		pending: { icon: ExclamationTriangle, class: 'text-warning', label: 'Waiting for DNS' },
		ready: { icon: GlobeAlt, class: 'text-success', label: 'HTTPS active' },
		error: { icon: ExclamationTriangle, class: 'text-destructive', label: 'Last check failed' }
	};

	/**
	 * A domain has two prerequisites and `status` only reports on one of them.
	 *
	 * The Traefik rule for a hostname lives in the container's labels, written at
	 * deploy time from the domains that existed then. A domain added since is
	 * invisible to the proxy however correct its DNS is: no router, so no
	 * certificate is ever requested, so the reconciler finds nothing and the row
	 * waits on DNS forever while pointing at the wrong prerequisite.
	 *
	 * Both facts are already here, so the panel can just say which step is
	 * outstanding.
	 */
	let lastDeployedAt = $derived(
		deployments.find((d) => d.status === 'success')?.created_at ?? null
	);

	// A deploy already on its way carries every domain added before it started, so
	// there is nothing to ask for — a prompt mid-flight would be wrong by the time
	// anyone finished reading it.
	let deployInFlight = $derived(deploying || deployments.some((d) => d.status === 'in_progress'));

	// Compared against when the deploy *started*, which is when the job snapshots
	// the domain list — a domain added a second later did not make that deploy.
	const needsDeploy = (domain: ServiceDomainResponse) =>
		!deployInFlight &&
		(!lastDeployedAt || Date.parse(domain.created_at) > Date.parse(lastDeployedAt));

	// Adding is visible in the list; removing is not, and it needs a deploy just
	// the same — so the session flag stays on as the half this cannot derive.
	let awaitingDeploy = $derived(needsRedeploy || domains.some(needsDeploy));

	/** Deploy, then get out of the way and let the reader watch it happen. */
	function deployFromDialog() {
		dnsDomainId = null;
		activeTab = 'deployments';
		deploy();
	}

	let envKey = $state('');
	let envValue = $state('');
	let envError = $state('');

	let loadToken = 0;

	async function loadDomains(id: string, token: number) {
		const { data } = await api.GET('/api/services/{id}/domains', {
			params: { path: { id } }
		});
		if (token !== loadToken) return;
		if (data) domains = data;
	}

	async function loadEnvs(id: string, token: number) {
		const { data, error } = await api.GET('/api/services/{id}/envs', {
			params: { path: { id } }
		});
		if (token !== loadToken) return;
		if (data) {
			envs = data;
			envsLoaded = true;
		} else if (error) {
			envsLoaded = false;
		}
	}

	async function loadDeployments(id: string, token: number = loadToken) {
		const { data } = await api.GET('/api/services/{id}/deployments', {
			params: { path: { id }, query: { limit: 10 } }
		});
		if (token !== loadToken) return;
		if (data) deployments = data;
	}

	async function deploy() {
		deployError = '';
		deploying = true;
		needsRedeploy = false;
		try {
			const { data, error } = await api.POST('/api/deployments', {
				body: { service_id: svcId }
			});
			if (error) {
				deployError = (error as { error: string }).error;
				return;
			}
			if (data) {
				localDeploymentId = data.deployment_id;
				loadDeployments(svcId);
			}
		} catch {
			deployError = 'Network error';
		} finally {
			deploying = false;
		}
	}

	async function addDomain() {
		domainError = '';
		domainAdding = true;
		try {
			const { data, error } = await api.POST('/api/services/{id}/domains', {
				params: { path: { id: svcId } },
				body: { domain: domainInput.trim() }
			});
			if (error) {
				domainError = (error as { error: string }).error;
				return;
			}
			if (data) {
				// No needsRedeploy here: an added domain is in the list, so the
				// derivation sees it. The flag is only for the half that leaves no
				// trace to derive from.
				domains = [...domains, data];
				domainInput = '';
				// The moment the name is accepted is the moment nobody knows what to
				// do next, so the answer arrives unasked rather than waiting to be
				// looked for.
				dnsDomainId = data.id;
			}
		} catch {
			domainError = 'Network error';
		} finally {
			domainAdding = false;
		}
	}

	async function deleteDomain(domainId: string) {
		await api.DELETE('/api/services/{id}/domains/{domainId}', {
			params: { path: { id: svcId, domainId } }
		});
		domains = domains.filter((d) => d.id !== domainId);
		needsRedeploy = true;
	}

	async function addEnv() {
		envError = '';
		const { data, error } = await api.POST('/api/services/{id}/envs', {
			params: { path: { id: svcId } },
			body: { key: envKey, value: envValue }
		});
		if (error) {
			envError = (error as { error: string }).error;
			return;
		}
		if (data) {
			const idx = envs.findIndex((e) => e.key === data.key);
			if (idx >= 0) {
				envs[idx] = data;
				envs = [...envs];
			} else {
				envs = [...envs, data].sort((a, b) => a.key.localeCompare(b.key));
			}
			envKey = '';
			envValue = '';
		}
	}

	async function deleteEnv(key: string) {
		await api.DELETE('/api/services/{id}/envs/{key}', {
			params: { path: { id: svcId, key } }
		});
		envs = envs.filter((e) => e.key !== key);
	}

	async function deleteService() {
		deleteError = '';
		deleting = true;
		try {
			const id = svcId;
			const { error } = await api.DELETE('/api/services/{id}', { params: { path: { id } } });
			if (error) {
				deleteError = (error as { error: string }).error;
				return;
			}
			deleteOpen = false;
			onDeleted?.(id);
		} catch {
			deleteError = 'Network error';
		} finally {
			deleting = false;
		}
	}

	$effect(() => {
		api.GET('/api/servers').then(({ data }) => {
			if (data) servers = data;
		});
	});

	$effect(() => {
		const id = svcId;
		const token = ++loadToken;

		domains = [];
		envs = [];
		envsLoaded = false;
		deployments = [];
		openDeployment = null;
		localDeploymentId = null;
		deployError = '';
		deploying = false;
		needsRedeploy = false;

		activeTab = 'deployments';
		domainInput = '';
		domainError = '';
		domainAdding = false;
		dnsDomainId = null;
		envKey = '';
		envValue = '';
		envError = '';
		deleteOpen = false;
		deleting = false;
		deleteError = '';
		savedAt = 0;
		// untrack: this effect resets the whole panel and must fire only when the
		// selected service changes. Reading `service` normally would make every
		// save re-run it and throw the reader back to the Deployments tab.
		resetEditForm(untrack(() => service));

		loadDomains(id, token);
		loadEnvs(id, token);
		loadDeployments(id, token);
	});

	/**
	 * A domain settles without anyone touching it: the reconciler promotes it once
	 * its certificate shows up. A panel left open should say so on its own —
	 * watching a row stay amber while the thing has been working for ten minutes
	 * is how people learn not to trust the row.
	 *
	 * The API sends the moment of the next pass, so this waits for that moment
	 * instead of sampling blindly. One request per pass, landing just after the
	 * answer can change, rather than four that mostly find the same thing. The
	 * fetched response carries the following pass, which schedules the next wait —
	 * and a settled list carries none, which is how the loop stops.
	 */
	const checkGraceMs = 3_000;

	// Every unresolved domain gets the same moment: it is one reconciler pass, not
	// a schedule per domain. Whichever is first will do.
	let nextCheckAt = $derived(domains.find((d) => d.next_check_at)?.next_check_at ?? null);

	$effect(() => {
		const iso = nextCheckAt;
		if (!iso) return;
		const id = svcId;

		// A floor rather than a raw difference: a clock skewed behind the server's
		// would otherwise schedule at zero and spin.
		const wait = Math.max(1_000, Date.parse(iso) + checkGraceMs - Date.now());
		const timer = setTimeout(() => loadDomains(id, loadToken), wait);

		// Coming back to a tab shows current data rather than whatever was true
		// when it went to the background — a throttled background timer can be a
		// while late, and sitting out the rest of it is what makes someone reach
		// for reload.
		const onVisibilityChange = () => {
			if (!document.hidden) loadDomains(id, loadToken);
		};
		document.addEventListener('visibilitychange', onVisibilityChange);

		return () => {
			clearTimeout(timer);
			document.removeEventListener('visibilitychange', onVisibilityChange);
		};
	});

	// One clock for the whole list: the countdown is to a single reconciler pass,
	// so every waiting row shows the same number.
	const nextCheck = countdownTo(() => nextCheckAt);

	const tabs: { id: Tab; label: string }[] = [
		{ id: 'deployments', label: 'Deployments' },
		{ id: 'logs', label: 'Logs' },
		{ id: 'domains', label: 'Domains' },
		{ id: 'env', label: 'Variables' },
		{ id: 'settings', label: 'Settings' }
	];

	let visibleTabs = $derived(tabs.filter((t) => t.id !== 'env' || (showEnvVars && canEdit)));
</script>

<!-- @container, not viewport breakpoints: this same component renders in the
     builder's inspector at clamp(420px, 55vw, 960px) and again on /services/[id]
     at a different width entirely. What the rows should do depends on the room
     *they* have, which the viewport does not know. -->
<div class={cn('@container flex h-full min-h-0 flex-col', className)}>
	<!-- Underline tabs, not filled pills. Four pills across a 420px panel put four
	     competing blocks above content that has none, and the row read as heavier
	     than the thing it was labelling. An underline marks one tab without
	     drawing a box around any of them. -->
	<div class="flex-none border-b border-border bg-card">
		<nav class="flex items-center gap-5 px-5" aria-label="Service sections">
			{#each visibleTabs as tab (tab.id)}
				<button
					type="button"
					onclick={() => (activeTab = tab.id)}
					class={cn(
						// -mb-px pulls the 2px marker onto the nav's own hairline; without it
						// the underline floats a pixel above the border and reads as a
						// misalignment rather than a join.
						'-mb-px cursor-pointer border-b-2 py-3.5 text-[15px] font-medium whitespace-nowrap transition-colors',
						'focus-visible:rounded-sm focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-card focus-visible:outline-none',
						activeTab === tab.id
							? 'border-foreground text-foreground'
							: 'border-transparent text-muted-foreground hover:text-foreground'
					)}
					aria-current={activeTab === tab.id ? 'page' : undefined}
				>
					{tab.label}
				</button>
			{/each}
		</nav>
	</div>

	<div class="min-h-0 flex-1 overflow-y-auto bg-card px-5 py-5">
		{#if activeTab === 'deployments'}
			<!-- Where the thing is reachable, and the one action that changes what's
			     running, on one line. The address is the reason you'd press Deploy, so
			     splitting them made you look in two places to decide one thing. -->
			<div class="flex items-center justify-between gap-3">
				<div class="flex min-w-0 items-center gap-1.5 text-[13px] text-muted-foreground">
					<Icon src={GlobeAlt} theme="outline" class="h-3.5 w-3.5 flex-none" />
					{#if primaryDomain}
						<a
							href="https://{primaryDomain.domain}"
							target="_blank"
							rel="noreferrer"
							class="truncate text-foreground hover:underline"
						>
							{primaryDomain.domain}
						</a>
						{#if domains.length > 1}
							<span class="flex-none">+{domains.length - 1}</span>
						{/if}
					{:else}
						<span class="truncate">No domain attached</span>
					{/if}
				</div>
				{#if canEdit}
					<Button onclick={deploy} loading={deploying} size="sm">
						{deploying ? 'Deploying...' : 'Deploy'}
					</Button>
				{/if}
			</div>

			<!-- Derived, not a session flag: the routing that is missing is missing
			     after a reload too, and a warning that vanished when you refreshed was
			     how this went unnoticed for five minutes at a time. -->
			{#if awaitingDeploy}
				<Alert tone="warning" class="mt-3 text-[13px]">
					Domain routing is not live yet. Deploy to apply it.
				</Alert>
			{/if}
			{#if deployError}
				<p class="mt-2 text-[15px] text-destructive">{deployError}</p>
			{/if}

			{#if latestDeployment || deploymentId}
				<!-- The current deployment and its output are one object, so they share
				     one card: the log panel drops its own edge and joins on below. Two
				     stacked outlines read as two unrelated things stacked by accident. -->
				<div class="mt-4 overflow-hidden rounded-lg border border-border">
					<!-- Narrow, the meta stacks under the image because there is no room for it
					     anywhere else. Once the panel is wide the same two facts sit on one
					     line, image left and meta right — which is what stops a 960px card
					     from being two short lines against half a metre of nothing. -->
					<div class="flex items-start gap-2.5 p-3 @2xl:items-center">
						<StatusBadge
							status={latestDeployment?.status ?? 'in_progress'}
							class="mt-px flex-none @2xl:mt-0"
						/>
						<div
							class="min-w-0 flex-1 @2xl:flex @2xl:items-baseline @2xl:justify-between @2xl:gap-6"
						>
							<p class="truncate font-mono text-[15px] text-foreground">{service.image}</p>
							<p class="mt-0.5 truncate text-[13px] text-muted-foreground @2xl:mt-0 @2xl:flex-none">
								{#if latestDeployment}
									<!-- Relative first: "2 hours ago" is what you actually want to know
									     about a deployment. The exact timestamp stays one hover away. -->
									<span title={formatDateTime(latestDeployment.created_at)}>
										{formatRelativeTime(latestDeployment.created_at)}
									</span>
									<span class="px-1">·</span>
									<span class="font-mono">{latestDeployment.id.slice(0, 8)}</span>
								{:else}
									Starting...
								{/if}
							</p>
						</div>
					</div>
					{#if deploymentId}
						<DeploymentLogs {deploymentId} flush onDone={() => loadDeployments(svcId)} />
					{/if}
				</div>
			{:else}
				<div class="mt-4 rounded-lg border border-dashed border-border">
					<EmptyState
						icon={Server}
						title="No deployments yet"
						description="Deploy this service to see its status, live logs and history here."
						class="px-5 py-8"
					/>
				</div>
			{/if}

			{#if previousDeployments.length > 0}
				<!-- mt-8, not mt-5: history is a different subject from what is running
				     now, and at 20px it sat closer to the current card than the card's own
				     rows sit to each other — proximity said "same thing" while the label
				     said otherwise. -->
				<div class="mt-8">
					<p class="mb-2 text-[13px] text-muted-foreground">Previous</p>
					<div class="overflow-hidden rounded-lg border border-border">
						{#each previousDeployments as dep (dep.id)}
							<button
								type="button"
								onclick={() => (openDeployment = dep)}
								class={cn(
									dataRowVariants({ density: 'dense', interactive: true }),
									'cursor-pointer gap-2.5 text-left text-[13px]'
								)}
							>
								<StatusBadge status={dep.status} class="flex-none" />
								<span class="truncate font-mono text-muted-foreground">{dep.id.slice(0, 8)}</span>
								<span
									class="ml-auto flex-none text-muted-foreground"
									title={formatDateTime(dep.created_at)}
								>
									{formatRelativeTime(dep.created_at)}
								</span>
							</button>
						{/each}
					</div>
				</div>
			{/if}
		{:else if activeTab === 'logs'}
			<!-- Gated on has_deployed rather than letting the stream fail: with no
			     container there is nothing to follow, and the endpoint's 400 would
			     only make EventSource retry it forever. -->
			{#if service.has_deployed}
				<LogStream endpoint="/api/services/{svcId}/logs" class="h-full" />
			{:else}
				<EmptyState
					icon={Server}
					title="No logs yet"
					description="Deploy this service to follow its container output here."
					class="px-5 py-8"
				/>
			{/if}
		{:else if activeTab === 'domains'}
			{#if domains.length === 0}
				<div class="rounded-lg border border-dashed border-border px-4 py-5 text-center">
					<p class="text-[15px] text-muted-foreground">No domains attached</p>
					<p class="mt-1 text-[13px] text-muted-foreground">
						Add one below to reach this service by name instead of by port.
					</p>
				</div>
			{:else}
				<!-- One divided card, not a stack of separate bordered pills: these are
				     rows of the same list, and giving each its own edge made a three-domain
				     panel read as three unrelated objects. -->
				<div class="overflow-hidden rounded-lg border border-border">
					{#each domains as domain (domain.id)}
						{@const waiting = domain.status !== 'ready' && needsDeploy(domain)}
						{@const mark = domainMarks[domain.status] ?? domainMarks.pending}
						<DataRow density="dense" class="items-start gap-2.5">
							<!-- The state leads the row instead of trailing it as a badge. It is
							     the first thing you want to know about an address, and reading it
							     in the margin beats hunting for a word on the far side — which is
							     also what frees the second line to say what to do about it. -->
							<Icon
								src={mark.icon}
								theme="outline"
								class={cn('mt-0.5 h-4 w-4 flex-none', mark.class)}
								aria-hidden="true"
							/>
							<div class="min-w-0 flex-1">
								<div class="flex min-w-0 items-center gap-2">
									<a
										href="https://{domain.domain}"
										target="_blank"
										rel="noreferrer"
										class="truncate text-[15px] font-medium text-foreground hover:underline"
									>
										{domain.domain}
									</a>
									{#if domain.is_primary}
										<Badge tone="info" class="flex-none">primary</Badge>
									{/if}
								</div>
								<!-- State and the way out of it on one line: what is wrong and what
								     to press about it belong next to each other, not stacked as two
								     separate remarks. -->
								<div class="mt-0.5 flex min-w-0 flex-wrap items-center gap-x-1.5 text-[13px]">
									{#if waiting}
										<!-- The prerequisite that is actually outstanding. Saying
										     "Waiting for DNS" to someone whose DNS is already correct is
										     what sent this panel's reader off to check a record that was
										     never the problem. -->
										<span class="min-w-0 truncate text-muted-foreground">Not deployed yet</span>
										<span class="flex-none text-muted-foreground/50" aria-hidden="true">·</span>
										<button
											type="button"
											onclick={deploy}
											class="flex-none cursor-pointer rounded font-medium text-primary-deep transition-colors hover:underline focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
										>
											Deploy
										</button>
									{:else}
										<span
											class={cn(
												'min-w-0 truncate',
												domain.status === 'error' ? 'text-destructive' : 'text-muted-foreground'
											)}
											title={domain.status === 'error' ? (domain.last_error ?? '') : undefined}
										>
											{domain.status === 'error' && domain.last_error
												? domain.last_error
												: mark.label}
										</span>
									{/if}
									<!-- Only while it is still waiting: once the certificate is issued
									     the record is demonstrably correct, and a link to instructions
									     for something already done is just noise on the row. -->
									{#if domain.status !== 'ready'}
										<!-- No countdown while the deploy is the missing piece: that
										     check cannot succeed yet, and counting down to it is the
										     exact reassurance that wasted somebody's afternoon. -->
										{#if !waiting && nextCheck.seconds !== null}
											<span class="flex-none text-muted-foreground/50" aria-hidden="true">·</span>
											<!-- Counting to the reconciler's own pass, not to a refresh of
											     ours: that is the moment the answer can change. A number
											     running out four times a minute while nothing happened
											     would teach the reader to stop looking at it.

											     Hidden from screen readers: a digit changing every second
											     is unusable read aloud, and the state itself is announced
											     by the text beside it the moment it arrives. -->
											<span class="flex-none tabular-nums" aria-hidden="true">
												{nextCheck.seconds > 0
													? `next check in ${nextCheck.seconds}s`
													: 'checking now…'}
											</span>
										{/if}
										<span class="flex-none text-muted-foreground/50" aria-hidden="true">·</span>
										<button
											type="button"
											onclick={() => (dnsDomainId = domain.id)}
											class="flex-none cursor-pointer rounded font-medium text-primary-deep transition-colors hover:underline focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
										>
											Show DNS record
										</button>
									{/if}
								</div>
							</div>
							{#if canEdit}
								<IconButton
									variant="ghost"
									onclick={() => deleteDomain(domain.id)}
									class="-mr-1 flex-none hover:text-destructive"
									aria-label="Remove {domain.domain}"
								>
									<Icon src={XMark} theme="outline" class="h-3.5 w-3.5" />
								</IconButton>
							{/if}
						</DataRow>
					{/each}
				</div>
			{/if}

			{#if canEdit}
				<form
					onsubmit={(e) => {
						e.preventDefault();
						addDomain();
					}}
					class="mt-4 flex items-end gap-2"
				>
					<FormField label="Add domain">
						<Input type="text" bind:value={domainInput} placeholder="myapp.example.com" required />
					</FormField>
					<Button type="submit" size="sm" loading={domainAdding}>
						{domainAdding ? 'Adding...' : 'Add'}
					</Button>
				</form>
				{#if domainError}
					<p class="mt-1 text-[15px] text-destructive">{domainError}</p>
				{/if}
				<!-- One line where a two-case list used to be. The list was the generic
				     version of what the dialog now says exactly — with the actual record
				     name and the actual server address — so keeping both meant saying it
				     twice and saying it vaguely first. This only has to stop someone from
				     expecting the domain to work the moment they press Add. -->
				<p class="mt-2 text-[13px] text-muted-foreground">
					A domain needs a DNS record before it resolves. You'll get the exact one to add next.
				</p>
			{/if}
		{:else if activeTab === 'env'}
			{#if canEdit && envsLoaded}
				<form
					onsubmit={(e) => {
						e.preventDefault();
						addEnv();
					}}
					class="mb-4 flex items-center gap-2"
				>
					<!-- A third of the row for the key, the rest for the value: at 420px the
					     two inputs were splitting evenly and a value like a connection string
					     got the same room as `PORT`. -->
					<Input
						type="text"
						bind:value={envKey}
						placeholder="KEY"
						required
						size="sm"
						class="w-28 flex-none font-mono"
					/>
					<Input
						type="text"
						bind:value={envValue}
						placeholder="value"
						required
						size="sm"
						class="min-w-0 flex-1 font-mono"
					/>
					<Button type="submit" size="sm" class="flex-none">Set</Button>
				</form>

				{#if envError}
					<p class="mb-2 text-[15px] text-destructive">{envError}</p>
				{/if}

				{#if envs.length === 0}
					<div class="rounded-lg border border-dashed border-border px-4 py-5 text-center">
						<p class="text-[15px] text-muted-foreground">No variables set</p>
						<p class="mt-1 text-[13px] text-muted-foreground">
							They are passed to the container on the next deployment.
						</p>
					</div>
				{:else}
					<!-- Key over value, not key = value on one line: a 420px panel wrapped
					     every real secret mid-string, and the wrapped tail lined up under
					     the key it did not belong to. Stacked, the key stays scannable and
					     the value gets the full width to break in. -->
					<div class="overflow-hidden rounded-lg border border-border">
						{#each envs as env (env.key)}
							<DataRow density="dense" class="items-start gap-2.5">
								<div class="min-w-0 flex-1 font-mono">
									<p class="truncate text-[15px] font-medium text-foreground">{env.key}</p>
									<p class="mt-0.5 text-[13px] break-all text-muted-foreground">{env.value}</p>
								</div>
								<IconButton
									variant="ghost"
									onclick={() => deleteEnv(env.key)}
									class="-mr-1 flex-none hover:text-destructive"
									aria-label="Remove {env.key}"
								>
									<Icon src={XMark} theme="outline" class="h-3.5 w-3.5" />
								</IconButton>
							</DataRow>
						{/each}
					</div>
				{/if}
			{:else}
				<p class="text-[15px] text-muted-foreground">
					Environment variables are only visible to workspace owners and developers.
				</p>
			{/if}
		{:else if activeTab === 'settings'}
			<!-- Sentence-case labels in a fixed left column, not four uppercase headings
			     over four short values. Uppercase + medium weight gave the labels more
			     visual weight than the data they name, which is backwards. `Kind` is
			     gone entirely: the API only accepts "application", so the row could
			     never say anything else. The card is what turns four floating pairs
			     into one block you can read as "this is the service". -->
			{#if canEdit}
				<form
					onsubmit={(e) => {
						e.preventDefault();
						saveService();
					}}
					class="flex flex-col gap-3"
				>
					<FormField label="Name">
						<Input type="text" bind:value={editName} size="sm" required />
					</FormField>

					<FormField label="Image">
						<Input type="text" bind:value={editImage} size="sm" required class="font-mono" />
					</FormField>

					<div class="flex flex-col gap-1.5">
						<label class="flex items-center gap-2 text-[15px] text-foreground">
							<input type="checkbox" bind:checked={editExposed} class="publish-toggle" />
							Publish to the internet
						</label>
						{#if !editExposed}
							<p class="text-[13px] text-muted-foreground">
								Only other services in this project can reach it.
							</p>
						{:else if domains.length > 0}
							<p class="text-[13px] text-muted-foreground">
								Domain traffic goes through the Uploy proxy — the host port below only applies once
								every domain is removed.
							</p>
						{/if}
					</div>

					<!-- Side by side because they are one decision read left to right:
					     reached here, answered there. Stacked, they read as two unrelated
					     numbers, which is exactly the confusion that made a service look
					     configured when it could never answer. Unpublished there is no left
					     half to read, so the field goes away rather than sitting there
					     disabled with a number nothing is listening on. -->
					<div class="flex gap-3">
						{#if editExposed}
							<div class="min-w-0 flex-1">
								<FormField label="Reachable on port">
									<Input
										type="number"
										bind:value={editHostPort}
										min={1}
										max={65535}
										size="sm"
										required
									/>
								</FormField>
							</div>
						{/if}
						<div class="min-w-0 flex-1">
							<FormField label="Listens on port">
								<Input
									type="number"
									bind:value={editContainerPort}
									min={1}
									max={65535}
									size="sm"
									required
								/>
							</FormField>
						</div>
					</div>

					{#if saveError}
						<p class="text-[13px] text-destructive">{saveError}</p>
					{/if}

					<div class="flex items-center gap-3">
						<Button type="submit" size="sm" loading={saving} disabled={saving || !edited}>
							{saving ? 'Saving...' : 'Save changes'}
						</Button>
						{#if edited}
							<span class="text-[13px] text-muted-foreground">
								Takes effect on the next deployment.
							</span>
						{:else if savedAt}
							<span class="text-[13px] text-success">Saved — deploy to apply.</span>
						{/if}
					</div>
				</form>
			{/if}

			<dl class={cn('overflow-hidden rounded-lg border border-border', canEdit && 'mt-4')}>
				{#each fixedMetadata as row (row.label)}
					<DataRow density="dense" class="gap-4 text-[15px]">
						<dt class="w-20 flex-none text-muted-foreground">{row.label}</dt>
						<dd class={cn('min-w-0 flex-1 truncate text-foreground', row.mono && 'font-mono')}>
							{row.value}
						</dd>
					</DataRow>
				{/each}
			</dl>

			{#if isOwner}
				<div class="mt-5">
					<p class="mb-2 text-[13px] text-muted-foreground">Danger zone</p>
					<!-- Tinted edge, plain surface: the border is enough to say "read this
					     twice", and a filled red block for an action nobody has taken yet
					     reads as an error that already happened. -->
					<div
						class="flex items-start justify-between gap-3 rounded-lg border border-destructive/25 p-3"
					>
						<div class="min-w-0">
							<p class="text-[15px] font-medium text-foreground">Delete service</p>
							<p class="mt-0.5 text-[13px] text-muted-foreground">
								Removes this service from Uploy and stops its container on the server.
							</p>
						</div>
						<Button
							variant="destructive"
							size="sm"
							class="flex-none"
							onclick={() => (deleteOpen = true)}
						>
							Delete
						</Button>
					</div>
				</div>
			{/if}
		{/if}
	</div>
</div>

<DnsRecordDialog
	domain={dnsDomain}
	serverHost={server?.host}
	needsDeploy={!!dnsDomain && needsDeploy(dnsDomain)}
	onDeploy={deployFromDialog}
	onClose={() => (dnsDomainId = null)}
/>

<Dialog bind:open={deleteOpen}>
	<DialogContent>
		<DialogHeader>
			<DialogTitle>Delete {service.name}?</DialogTitle>
		</DialogHeader>
		<div class="px-5 pb-5 text-sm text-muted-foreground">
			Its container is stopped and removed from the server, and its domains and environment
			variables go with it. This cannot be undone.
			{#if deleteError}
				<p class="mt-2 text-destructive">{deleteError}</p>
			{/if}
		</div>
		<DialogFooter>
			<Button type="button" variant="secondary" size="sm" onclick={() => (deleteOpen = false)}>
				Cancel
			</Button>
			<Button
				type="button"
				variant="destructive"
				size="sm"
				loading={deleting}
				onclick={deleteService}
			>
				{deleting ? 'Deleting...' : 'Delete'}
			</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>

<style>
	/* Native input, sized and tinted to the design system rather than replaced by
	   a custom control: the platform already gives the keyboard, the focus ring
	   and the announcement for free. */
	.publish-toggle {
		width: 1rem;
		height: 1rem;
		accent-color: var(--primary);
		cursor: pointer;
	}
</style>
