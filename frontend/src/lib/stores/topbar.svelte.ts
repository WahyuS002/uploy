import type { IconSource } from '@steeze-ui/svelte-icon';

export type TopbarState = {
	title: string;
	icon?: IconSource;
};

class TopbarStore {
	state = $state<TopbarState>({ title: '' });

	set(next: TopbarState) {
		this.state = next;
	}

	clear() {
		this.state = { title: '' };
	}
}

export const topbar = new TopbarStore();
