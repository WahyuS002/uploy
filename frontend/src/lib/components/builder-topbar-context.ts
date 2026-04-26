import { getContext, setContext, type Snippet } from 'svelte';

export type BuilderTopbarState = {
	label: string;
	leading: Snippet | null;
	action: Snippet | null;
};

const KEY = Symbol('builder-topbar');

export function provideBuilderTopbar(state: BuilderTopbarState): BuilderTopbarState {
	setContext(KEY, state);
	return state;
}

export function useBuilderTopbar(): BuilderTopbarState | null {
	return getContext<BuilderTopbarState | null>(KEY) ?? null;
}
