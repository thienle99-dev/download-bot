import { writable } from 'svelte/store';
import type { LogMessage } from '../types';
import { BASE_URL, getToken } from '../api';

export const logsStore = writable<LogMessage[]>([]);
export const logConnectionState = writable<'disconnected' | 'connecting' | 'connected'>('disconnected');

let socket: WebSocket | null = null;
let reconnectTimeout: any = null;
let active = false;

export function connectLogs() {
  active = true;
  if (socket && socket.readyState === WebSocket.OPEN) {
    return;
  }

  // Parse WebSocket URL from BASE_URL
  const token = getToken();
  if (!token) return;

  const wsScheme = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  let wsUrl = '';

  if (BASE_URL.startsWith('http')) {
    // Development environment
    const baseWithoutHttp = BASE_URL.replace(/^https?:\/\//, '');
    wsUrl = `${wsScheme}//${baseWithoutHttp}/ws?token=${encodeURIComponent(token)}`;
  } else {
    // Production environment
    wsUrl = `${wsScheme}//${window.location.host}${BASE_URL}/ws?token=${encodeURIComponent(token)}`;
  }

  logConnectionState.set('connecting');
  
  try {
    socket = new WebSocket(wsUrl);

    socket.onopen = () => {
      logConnectionState.set('connected');
    };

    socket.onmessage = (event) => {
      try {
        const msg: LogMessage = JSON.parse(event.data);
        logsStore.update((current) => {
          const next = [...current, msg];
          // Limit to max 500 lines of logs to protect memory
          if (next.length > 500) {
            return next.slice(next.length - 500);
          }
          return next;
        });
      } catch (err) {
        console.error('Failed to parse websocket log message:', err);
      }
    };

    socket.onclose = () => {
      logConnectionState.set('disconnected');
      if (active) {
        // Attempt reconnect after 3 seconds
        clearTimeout(reconnectTimeout);
        reconnectTimeout = setTimeout(connectLogs, 3000);
      }
    };

    socket.onerror = () => {
      socket?.close();
    };
  } catch (err) {
    console.error('WebSocket connection error:', err);
    logConnectionState.set('disconnected');
  }
}

export function disconnectLogs() {
  active = false;
  clearTimeout(reconnectTimeout);
  if (socket) {
    socket.close();
    socket = null;
  }
  logConnectionState.set('disconnected');
}

export function clearLogsStore() {
  logsStore.set([]);
}
