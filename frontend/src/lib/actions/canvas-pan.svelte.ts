type Bounds = { x: number; y: number } | 'auto' | null;

type CanvasPanOptions = {
	enabled?: () => boolean;
	bounds?: Bounds | (() => Bounds);
	ignoreSelector?: string;
	gutter?: number;
};

const DEFAULT_IGNORE = 'input, textarea, select, button, a, label, [data-no-pan]';
const DEFAULT_GUTTER = 96;
const ZOOM_STEP = 0.1;
const ZOOM_MIN = 0.8;
const ZOOM_MAX = 1.4;

export function createCanvasPan(options: CanvasPanOptions = {}) {
	let x = $state(0);
	let y = $state(0);
	let scale = $state(1);
	let isPanning = $state(false);

	const ignoreSelector = options.ignoreSelector ?? DEFAULT_IGNORE;
	const gutter = options.gutter ?? DEFAULT_GUTTER;

	function resolveBounds(node: HTMLElement): { x: number; y: number } | null {
		const raw = typeof options.bounds === 'function' ? options.bounds() : options.bounds;
		if (raw == null) return null;
		if (raw === 'auto') {
			const rect = node.getBoundingClientRect();
			return {
				x: Math.max(0, rect.width / 2 - gutter),
				y: Math.max(0, rect.height / 2 - gutter)
			};
		}
		return raw;
	}

	function clamp(value: number, max: number) {
		if (max <= 0) return 0;
		return Math.max(-max, Math.min(max, value));
	}

	function clampZoom(value: number) {
		return Math.max(ZOOM_MIN, Math.min(ZOOM_MAX, value));
	}

	function roundZoom(value: number) {
		return Math.round(value * 100) / 100;
	}

	function viewport(node: HTMLElement) {
		let activePointer: number | null = null;
		let startX = 0;
		let startY = 0;
		let originX = 0;
		let originY = 0;

		let pointerFine = true;
		let mql: MediaQueryList | null = null;
		const onMqChange = (event: MediaQueryListEvent) => {
			pointerFine = event.matches;
		};
		if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
			mql = window.matchMedia('(pointer: fine)');
			pointerFine = mql.matches;
			mql.addEventListener('change', onMqChange);
		}

		function isEnabled() {
			if (!pointerFine) return false;
			if (options.enabled && !options.enabled()) return false;
			return true;
		}

		function onPointerDown(event: PointerEvent) {
			if (!isEnabled()) return;
			if (event.button !== 0) return;
			const target = event.target as Element | null;
			if (target && target.closest(ignoreSelector)) return;

			activePointer = event.pointerId;
			startX = event.clientX;
			startY = event.clientY;
			originX = x;
			originY = y;
			isPanning = true;
			try {
				node.setPointerCapture(activePointer);
			} catch {
				// ignore — capture is best-effort
			}
		}

		function onPointerMove(event: PointerEvent) {
			if (activePointer === null || event.pointerId !== activePointer) return;
			const dx = event.clientX - startX;
			const dy = event.clientY - startY;
			const bounds = resolveBounds(node);
			const nextX = originX + dx;
			const nextY = originY + dy;
			if (bounds) {
				x = clamp(nextX, bounds.x);
				y = clamp(nextY, bounds.y);
			} else {
				x = nextX;
				y = nextY;
			}
		}

		function endPan(event: PointerEvent) {
			if (activePointer === null || event.pointerId !== activePointer) return;
			try {
				node.releasePointerCapture(activePointer);
			} catch {
				// ignore
			}
			activePointer = null;
			isPanning = false;
		}

		node.addEventListener('pointerdown', onPointerDown);
		node.addEventListener('pointermove', onPointerMove);
		node.addEventListener('pointerup', endPan);
		node.addEventListener('pointercancel', endPan);

		return {
			destroy() {
				node.removeEventListener('pointerdown', onPointerDown);
				node.removeEventListener('pointermove', onPointerMove);
				node.removeEventListener('pointerup', endPan);
				node.removeEventListener('pointercancel', endPan);
				mql?.removeEventListener('change', onMqChange);
			}
		};
	}

	return {
		viewport,
		get x() {
			return x;
		},
		get y() {
			return y;
		},
		get scale() {
			return scale;
		},
		get isPanning() {
			return isPanning;
		},
		zoomIn() {
			scale = roundZoom(clampZoom(scale + ZOOM_STEP));
		},
		zoomOut() {
			scale = roundZoom(clampZoom(scale - ZOOM_STEP));
		},
		resetZoom() {
			scale = 1;
		},
		recenter() {
			x = 0;
			y = 0;
		},
		reset() {
			x = 0;
			y = 0;
			scale = 1;
		}
	};
}
