import { createApiClient } from '$lib/api/client';
import type { components } from '$lib/api/v1';

type SSHKeyResponse = components['schemas']['SSHKeyResponse'];
type ServerResponse = components['schemas']['ServerResponse'];

export const CREATE_NEW_SSH_KEY = '__create_new__';

type Options = {
	onSuccess?: (server: ServerResponse) => void;
};

export class ServerCreateController {
	private api = createApiClient();
	private onSuccess?: (server: ServerResponse) => void;

	keys = $state<SSHKeyResponse[]>([]);
	keysLoading = $state(false);
	keysLoaded = $state(false);
	keysError = $state('');
	sshKeyDialogOpen = $state(false);

	name = $state('');
	host = $state('');
	port = $state(22);
	sshUser = $state('root');
	sshKeyId = $state('');
	selectValue = $state('');
	error = $state('');
	loading = $state(false);

	checking = $state(false);
	verified = $state<{ host: string; port: number; sshUser: string; sshKeyId: string } | null>(null);

	selectedKeyPublicKey = $derived(this.keys.find((k) => k.id === this.sshKeyId)?.public_key ?? '');

	isVerified = $derived(
		this.verified !== null &&
			this.verified.host === this.host &&
			this.verified.port === this.port &&
			this.verified.sshUser === this.sshUser &&
			this.verified.sshKeyId === this.sshKeyId
	);

	canCheckConnection = $derived(
		this.host.trim() !== '' &&
			this.sshUser.trim() !== '' &&
			this.sshKeyId !== '' &&
			!this.checking &&
			!this.keysError
	);

	keyItems = $derived([
		{ value: CREATE_NEW_SSH_KEY, label: 'Create new SSH key' },
		...this.keys.map((k) => ({ value: k.id, label: k.name }))
	]);

	constructor(opts: Options = {}) {
		this.onSuccess = opts.onSuccess;
	}

	reset = () => {
		this.name = '';
		this.host = '';
		this.port = 22;
		this.sshUser = 'root';
		this.sshKeyId = '';
		this.selectValue = '';
		this.error = '';
		this.verified = null;
		this.checking = false;
		this.loading = false;
		this.sshKeyDialogOpen = false;
	};

	loadKeys = async (force = false) => {
		if (this.keysLoading) return;
		if (this.keysLoaded && !force && !this.keysError) return;
		this.keysLoading = true;
		this.keysError = '';
		try {
			const res = await this.api.GET('/api/ssh-keys');
			if (res.error) {
				this.keysError = (res.error as { error: string }).error ?? 'Failed to load SSH keys';
				return;
			}
			if (res.data) this.keys = res.data;
			this.keysLoaded = true;
		} catch {
			this.keysError = 'Network error loading SSH keys';
		} finally {
			this.keysLoading = false;
		}
	};

	handleKeySelectChange = (value: string) => {
		if (value === CREATE_NEW_SSH_KEY) {
			this.sshKeyDialogOpen = true;
			this.selectValue = this.sshKeyId;
			return;
		}
		this.selectValue = value;
		this.sshKeyId = value;
	};

	handleKeyCreated = async (key: SSHKeyResponse) => {
		this.sshKeyDialogOpen = false;
		await this.loadKeys(true);
		this.sshKeyId = key.id;
		this.selectValue = key.id;
	};

	checkConnection = async () => {
		this.error = '';
		this.checking = true;
		this.verified = null;
		try {
			const { error: err } = await this.api.POST('/api/servers/check-connection', {
				body: {
					host: this.host,
					port: this.port,
					ssh_user: this.sshUser,
					ssh_key_id: this.sshKeyId
				}
			});
			if (err) {
				this.error = (err as { error: string }).error;
				return;
			}
			this.verified = {
				host: this.host,
				port: this.port,
				sshUser: this.sshUser,
				sshKeyId: this.sshKeyId
			};
		} catch {
			this.error = 'Network error, please try again';
		} finally {
			this.checking = false;
		}
	};

	createServer = async () => {
		if (!this.isVerified) return;
		this.error = '';
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
				this.error = (err as { error: string }).error;
				return;
			}
			this.name = '';
			this.host = '';
			this.port = 22;
			this.sshUser = 'root';
			this.sshKeyId = '';
			this.selectValue = '';
			this.verified = null;
			if (data) this.onSuccess?.(data);
		} catch {
			this.error = 'Network error, please try again';
		} finally {
			this.loading = false;
		}
	};
}
