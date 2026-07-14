import { decodeMulti, ExtensionCodec } from "@msgpack/msgpack";
import { CircularLogBuffer } from "../utils/CircularLogBuffer";
import type { FlatLogRecord } from "../types/Log";
import type { ConnectionStatus } from "../types/Connection";

// Create a custom extension codec to handle Go's msgp time.Time (type 5)
const extensionCodec = new ExtensionCodec();
extensionCodec.register({
    type: 5,
    encode: () => null,
    decode: (data: Uint8Array) => {
        const view = new DataView(data.buffer, data.byteOffset, data.byteLength);

        // Bytes 0-7: 64-bit Big-Endian Unix seconds
        const seconds = Number(view.getBigInt64(0, false));
        // Bytes 8-11: 32-bit Big-Endian nanoseconds
        const nanos = view.getUint32(8, false);

        // convert to unix timestamp in milliseconds
        return seconds * 1000 + Math.floor(nanos / 1_000_000);
    },
});

const MAX_GLOBAL_LOGS = 300;
const RENDER_INTERVAL_MS = 1000;
const WEBSOCKET_NORMAL_CLOSURE = 1000;
const logBuffer = new CircularLogBuffer<FlatLogRecord>(MAX_GLOBAL_LOGS);

let ws: WebSocket | null = null;
let isPaused = false;
let reconnectAttempts = 0;
let maxAttempts = 5;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

// Helper to notify React of connection changes
function updateStatus(status: ConnectionStatus) {
    postMessage({ type: "STATUS", payload: status });
}

function connect() {
    if (ws?.readyState === WebSocket.OPEN) return;

    updateStatus("connecting");
    ws = new WebSocket("ws://localhost:8091/ws/logs/tail");

    ws.onopen = () => {
        reconnectAttempts = 0; // reset backoff on successful connect
        updateStatus("live");
    };

    ws.onmessage = async (e) => {
        if (!e.data) return; // ignore empty messages
        try {
            const buf = e.data instanceof Blob ? await e.data.arrayBuffer() : e.data;
            // decodeMulti parses the concatenated byte stream into individual objects
            for (const record of decodeMulti(buf, { extensionCodec })) {
                logBuffer.add(record as FlatLogRecord);
            }
        } catch (err) {
            console.error("Worker: failed to parse msgpack", err);
        }
    };

    ws.onerror = (e) => {
        // TODO: comment out in prod
        console.error("Wroker: Websocket error", e);
    };

    ws.onclose = (e) => {
        ws = null;
        if (e.code === WEBSOCKET_NORMAL_CLOSURE) return;

        if (reconnectAttempts >= maxAttempts) {
            updateStatus("error");
            return;
        }

        updateStatus("connecting");
        // Exponential backoff: 1s, 2s, 4s, 8s, 16s
        const delay = Math.min(1000 * 2 ** reconnectAttempts, 30_000);
        reconnectAttempts += 1;

        if (reconnectTimer) clearTimeout(reconnectTimer);
        reconnectTimer = setTimeout(connect, delay);
    };
}

// Start initial connection
connect();

// Send to React thread every second
setInterval(() => {
    if (!isPaused) {
        postMessage({ type: "LOG_UPDATE", payload: logBuffer.toArrayNewestFirst() });
    }
}, RENDER_INTERVAL_MS);

self.onmessage = (e) => {
    if (e.data.type === "PAUSE") {
        isPaused = true;
    }

    if (e.data.type === "RESUME") {
        isPaused = false;
        postMessage({ type: "LOG_UPDATE", payload: logBuffer.toArrayNewestFirst() });
    }

    if (e.data.type === "RECONNECT") {
        reconnectAttempts = 0;
        if (reconnectTimer) clearTimeout(reconnectTimer);
        connect();
    }

    if (e.data.type === "CLEANUP") {
        if (reconnectTimer) clearTimeout(reconnectTimer);
        ws?.close(WEBSOCKET_NORMAL_CLOSURE);
    }
};
