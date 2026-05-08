import { create } from "zustand";

export type WsConnectionStatus = "connected" | "connecting" | "reconnecting" | "disconnected";

export interface WsEventProgress {
  type: "progress";
  query_id: string;
  progress?: string;
  percent?: number;
}

export interface WsEventDone {
  type: "done";
  query_id: string;
  data: { rows: Record<string, unknown>[]; columns: { name: string; type: string }[] };
}

export interface WsEventResultReady {
  type: "result_ready";
  query_id: string;
  download_url: string;
}

export interface WsEventError {
  type: "error";
  query_id: string;
  message: string;
}

export type WsEvent = WsEventProgress | WsEventDone | WsEventResultReady | WsEventError;

export type WsEventHandler = (event: WsEvent) => void;

interface WsConnection {
  ws: WebSocket | null;
  status: WsConnectionStatus;
  retryCount: number;
  fallbackToPolling: boolean;
  handler: WsEventHandler;
  cleanup: (() => void) | null;
}

interface WsStoreState {
  connections: Record<string, WsConnection>;
  subscribe: (queryId: string, handler: WsEventHandler) => void;
  unsubscribe: (queryId: string) => void;
  getStatus: (queryId: string) => WsConnectionStatus;
  isFallbackToPolling: (queryId: string) => boolean;
  reconnectBackoff: (attempt: number) => number;
}

const BACKOFF_DELAYS = [1000, 2000, 4000];
const MAX_RETRIES = 3;

function getToken(): string | null {
  try {
    const stored = localStorage.getItem("auth-storage");
    if (stored) {
      const parsed = JSON.parse(stored);
      return parsed?.state?.accessToken || null;
    }
  } catch {
    return null;
  }
  return null;
}

export const useWsStore = create<WsStoreState>((set, get) => ({
  connections: {},

  subscribe: (queryId: string, handler: WsEventHandler) => {
    const existing = get().connections[queryId];
    if (existing) {
      existing.cleanup?.();
      if (existing.ws) {
        existing.ws.onclose = null;
        existing.ws.close();
      }
    }

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const token = getToken();
    const wsUrl = token
      ? `${protocol}//${window.location.host}/ws/query/${queryId}?token=${token}`
      : `${protocol}//${window.location.host}/ws/query/${queryId}`;

    let ws: WebSocket | null = null;
    let retryCount = 0;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

    const connect = () => {
      ws = new WebSocket(wsUrl);
      const currentWs = ws;

      set(state => ({
        connections: {
          ...state.connections,
          [queryId]: {
            ...state.connections[queryId],
            ws: currentWs,
            status: "connecting" as WsConnectionStatus,
          },
        },
      }));

      currentWs.onopen = () => {
        retryCount = 0;
        set(state => ({
          connections: {
            ...state.connections,
            [queryId]: {
              ...state.connections[queryId],
              ws: currentWs,
              status: "connected" as WsConnectionStatus,
              retryCount: 0,
              fallbackToPolling: false,
            },
          },
        }));
      };

      currentWs.onmessage = (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data) as WsEvent;
          handler(data);
        } catch {
          console.error("[wsStore] failed to parse message:", event.data);
        }
      };

      currentWs.onclose = () => {
        if (currentWs !== ws) return;

        if (retryCount < MAX_RETRIES) {
          retryCount++;
          const delay = get().reconnectBackoff(retryCount - 1);

          set(state => ({
            connections: {
              ...state.connections,
              [queryId]: {
                ...state.connections[queryId],
                ws: null,
                status: "reconnecting" as WsConnectionStatus,
                retryCount,
              },
            },
          }));

          reconnectTimer = setTimeout(() => {
            if (ws === null) return;
            connect();
          }, delay);
        } else {
          set(state => ({
            connections: {
              ...state.connections,
              [queryId]: {
                ...state.connections[queryId],
                ws: null,
                status: "disconnected" as WsConnectionStatus,
                retryCount,
                fallbackToPolling: true,
              },
            },
          }));
        }
      };

      currentWs.onerror = () => {
        ws?.close();
      };
    };

    connect();

    const cleanup = () => {
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
      }
      if (ws) {
        ws.onclose = null;
        ws.close();
        ws = null;
      }
    };

    set(state => ({
      connections: {
        ...state.connections,
        [queryId]: {
          ws,
          status: "connecting" as WsConnectionStatus,
          retryCount: 0,
          fallbackToPolling: false,
          handler,
          cleanup,
        },
      },
    }));
  },

  unsubscribe: (queryId: string) => {
    const conn = get().connections[queryId];
    if (conn) {
      conn.cleanup?.();
    }
    set(state => {
      const next = { ...state.connections };
      delete next[queryId];
      return { connections: next };
    });
  },

  getStatus: (queryId: string): WsConnectionStatus => {
    return get().connections[queryId]?.status ?? "disconnected";
  },

  isFallbackToPolling: (queryId: string): boolean => {
    return get().connections[queryId]?.fallbackToPolling ?? false;
  },

  reconnectBackoff: (attempt: number): number => {
    return BACKOFF_DELAYS[attempt] ?? 4000;
  },
}));
