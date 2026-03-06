import { get } from 'svelte/store';
import { auth } from '$lib/stores/auth';
import { silentRefresh } from '$lib/api/client';

export type SSEMessageHandler = (data: string) => void;
export type SSEErrorHandler = (err: Error) => void;

export function openSSEStream(
	url: string,
	onMessage: SSEMessageHandler,
	onError: SSEErrorHandler,
	onClose?: () => void
): AbortController {
	const controller = new AbortController();

	(async () => {
		const fetchStream = async (): Promise<Response> => {
			let token = get(auth).accessToken;
			if (!token) {
				await silentRefresh();
				token = get(auth).accessToken;
			}
			if (!token) {
				throw new Error('not authenticated');
			}
			let res = await fetch(url, {
				signal: controller.signal,
				headers: {
					Authorization: `Bearer ${token}`,
					Accept: 'text/event-stream'
				}
			});
			if (res.status !== 401) {
				return res;
			}
			await silentRefresh();
			token = get(auth).accessToken;
			if (!token) {
				throw new Error('not authenticated');
			}
			res = await fetch(url, {
				signal: controller.signal,
				headers: {
					Authorization: `Bearer ${token}`,
					Accept: 'text/event-stream'
				}
			});
			return res;
		};

		let res: Response;
		try {
			res = await fetchStream();
		} catch (e) {
			if (!controller.signal.aborted) {
				onError(e instanceof Error ? e : new Error(String(e)));
			}
			return;
		}

		if (!res.ok || !res.body) {
			onError(new Error(`SSE request failed: ${res.status}`));
			return;
		}

		const reader = res.body.getReader();
		const decoder = new TextDecoder();
		let buf = '';

		try {
			while (true) {
				const { done, value } = await reader.read();
				if (done) {
					if (!controller.signal.aborted) {
						onClose?.();
					}
					break;
				}
				buf += decoder.decode(value, { stream: true });
				const lines = buf.split('\n');
				buf = lines.pop() ?? '';
				let dataLine = '';
				for (const line of lines) {
					if (line.startsWith('data:')) {
						dataLine = line.slice(5).trim();
					} else if (line === '') {
						if (dataLine !== '') {
							onMessage(dataLine);
							dataLine = '';
						}
					}
				}
			}
		} catch (e) {
			if (!controller.signal.aborted) {
				onError(e instanceof Error ? e : new Error(String(e)));
			}
		}
	})();

	return controller;
}
