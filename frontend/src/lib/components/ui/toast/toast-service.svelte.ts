import type { IconSource } from '@steeze-ui/svelte-icon';
import type { Component } from 'svelte';

export type ToastTone = 'neutral' | 'success' | 'error';

export type ToastIcon =
	| { kind: 'heroicon'; src: IconSource }
	| { kind: 'lucide'; component: Component<{ class?: string; strokeWidth?: number }> }
	| { kind: 'spinner' };

export type ToastInput = {
	id?: string;
	tone?: ToastTone;
	title: string;
	description?: string;
	icon?: ToastIcon | null;
	dismissible?: boolean;
	duration?: number;
};

export type Toast = {
	id: string;
	tone: ToastTone;
	title: string;
	description: string;
	icon: ToastIcon | null;
	dismissible: boolean;
	duration: number;
	createdAt: number;
};

export const TOAST_QUEUE_LIMIT = 3;

class ToastStore {
	items = $state<Toast[]>([]);
	private nextId = 0;

	show(input: ToastInput): string {
		const id = input.id ?? `t-${++this.nextId}`;
		const next: Toast = {
			id,
			tone: input.tone ?? 'neutral',
			title: input.title,
			description: input.description ?? '',
			icon: input.icon ?? null,
			dismissible: input.dismissible ?? true,
			duration: input.duration ?? 0,
			createdAt: Date.now()
		};

		const existing = this.items.findIndex((t) => t.id === id);
		if (existing >= 0) {
			this.items[existing] = next;
			return id;
		}

		this.items.push(next);
		if (this.items.length > TOAST_QUEUE_LIMIT) {
			this.items.splice(0, this.items.length - TOAST_QUEUE_LIMIT);
		}
		return id;
	}

	dismiss(id: string): void {
		const idx = this.items.findIndex((t) => t.id === id);
		if (idx >= 0) this.items.splice(idx, 1);
	}

	clear(): void {
		this.items.length = 0;
	}
}

export const toastStore = new ToastStore();

export const toast = {
	show: (input: ToastInput) => toastStore.show(input),
	neutral: (input: Omit<ToastInput, 'tone'>) => toastStore.show({ ...input, tone: 'neutral' }),
	success: (input: Omit<ToastInput, 'tone'>) => toastStore.show({ ...input, tone: 'success' }),
	error: (input: Omit<ToastInput, 'tone'>) => toastStore.show({ ...input, tone: 'error' }),
	dismiss: (id: string) => toastStore.dismiss(id),
	clear: () => toastStore.clear()
};
