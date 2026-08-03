import { createApiClient } from '$lib/api/client';
import type { components } from '$lib/api/v1';

type SSHKeyResponse = components['schemas']['SSHKeyResponse'];
type ServerResponse = components['schemas']['ServerResponse'];
type ErrorResponse = components['schemas']['ErrorResponse'];
type FailureCode = NonNullable<ErrorResponse['code']>;

type Options = {
	onSuccess?: (server: ServerResponse) => void;
};

export type ServerCreateStep = 'target' | 'authorize';

/** A failed connection attempt, kept as a stage rather than a string so the UI
 *  can offer the remedy that matches. */
export type ConnectFailure = {
	code: FailureCode | 'unknown';
	message: string;
	/** True when the fix lives on the target step, not the authorize step. */
	belongsToTarget: boolean;
};

const TARGET_FAILURES: ReadonlySet<string> = new Set(['unreachable']);

export class ServerCreateController {
	private api = createApiClient();
	private onSuccess?: (server: ServerResponse) => void;

	keys = $state<SSHKeyResponse[]>([]);
	keysLoading = $state(false);
	keysLoaded = $state(false);
	keysError = $state('');
	sshKeyDialogOpen = $state(false);

	step = $state<ServerCreateStep>('target');

	name = $state('');
	host = $state('');
	port = $state(22);
	sshUser = $state('root');
	sshKeyId = $state('');
	failure = $state<ConnectFailure | null>(null);
	loading = $state(false);

	/** Advancing to the authorize step costs nothing, so it only needs the
	 *  fields that the authorize instructions interpolate. */
	canAdvance = $derived(
		this.name.trim() !== '' &&
			this.host.trim() !== '' &&
			this.sshUser.trim() !== '' &&
			this.sshKeyId !== '' &&
			!this.keysError
	);

	selectedKey = $derived(this.keys.find((k) => k.id === this.sshKeyId));

	/** The exact line to run on the target host: real key, real user, no
	 *  placeholders left for the reader to substitute. */
	authorizeCommand = $derived(
		this.selectedKey?.public_key
			? `echo '${this.selectedKey.public_key.trim()}' >> ~/.ssh/authorized_keys`
			: ''
	);

	constructor(opts: Options = {}) {
		this.onSuccess = opts.onSuccess;
	}

	reset = () => {
		this.step = 'target';
		this.name = '';
		this.host = '';
		this.port = 22;
		this.sshUser = 'root';
		this.sshKeyId = '';
		this.failure = null;
		this.loading = false;
		this.sshKeyDialogOpen = false;
	};

	advance = () => {
		if (!this.canAdvance) return;
		this.failure = null;
		this.step = 'authorize';
	};

	back = () => {
		this.failure = null;
		this.step = 'target';
	};

	loadKeys = async (force = false) => {
		if (this.keysLoading) return;
		if (this.keysLoaded && !force && !this.keysError) return;
		this.keysLoading = true;
		this.keysError = '';
		try {
			const res = await this.api.GET('/api/ssh-keys');
			if (res.error) {
				this.keysError = (res.error as ErrorResponse).error ?? 'Failed to load SSH keys';
				return;
			}
			if (res.data) this.keys = res.data;
			this.keysLoaded = true;
			// Only one key to pick from: preselect it so the public key is visible right away.
			if (!this.sshKeyId && this.keys.length === 1) this.sshKeyId = this.keys[0].id;
		} catch {
			this.keysError = 'Network error loading SSH keys';
		} finally {
			this.keysLoading = false;
		}
	};

	handleKeyCreated = async (key: SSHKeyResponse) => {
		this.sshKeyDialogOpen = false;
		await this.loadKeys(true);
		this.sshKeyId = key.id;
	};

	/** Submitting *is* the connection test: the API probes SSH, session and
	 *  docker before it writes a record, so an unreachable host never lands. */
	createServer = async () => {
		this.failure = null;
		this.loading = true;
		try {
			const { data, error: err } = await this.api.POST('/api/servers', {
				body: {
					name: this.name,
					host: this.host,
					port: this.port,
					ssh_user: this.sshUser,
					ssh_key_id: this.sshKeyId
				}
			});
			if (err) {
				const body = err as ErrorResponse;
				const code = body.code ?? 'unknown';
				this.failure = {
					code,
					message: body.error,
					belongsToTarget: TARGET_FAILURES.has(code)
				};
				return;
			}
			if (data) this.onSuccess?.(data);
		} catch {
			this.failure = {
				code: 'unknown',
				message: 'Network error, please try again',
				belongsToTarget: false
			};
		} finally {
			this.loading = false;
		}
	};
}
