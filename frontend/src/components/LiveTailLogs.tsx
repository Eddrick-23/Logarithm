import {
    Box,
    Typography,
    Stack,
    Button,
    type SelectChangeEvent,
    CircularProgress,
    TableContainer,
    Table,
    TableHead,
    TableRow,
    TableCell,
    TableBody,
} from "@mui/material";
import { useState, useRef, useEffect, useCallback, useMemo } from "react";
import { card, pulseSx, sectionLabel } from "../theme/tokens";
import PauseIcon from "@mui/icons-material/Pause";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import type { FlatLogRecord, LogType } from "../types/Log";
import { useDistinctServices } from "../hooks/useDistinctServices";
import ErrorBanner from "./ErrorBanner";
import { decodeMulti, ExtensionCodec } from "@msgpack/msgpack";
import { parseSeverity } from "../utils/severity";
import TailLogRow from "./TailLogRow";
import SearchField from "./SearchField";
import SeverityDropdown from "./SeverityDropdown";
import ServiceDropdown from "./ServiceDropdown";

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

type ConnectionStatus = "connecting" | "connected" | "error";

const MAX_GLOBAL_LOGS = 300;
const MAX_DISPLAY_LOGS = 15;
const WEBSOCKET_NORMAL_CLOSURE = 1000;

export default function LiveTailLogs() {
    const wsRef = useRef<WebSocket | null>(null);
    const reconnectAttempts = useRef(0);
    const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
    const [logs, setLogs] = useState<FlatLogRecord[]>([]);
    const [severities, setSeverities] = useState<LogType[]>([]); // empty indicates all severities selected
    const [services, setServices] = useState<string[]>([]); // empty indicates all services selected
    const [debouncedSearch, setDebouncedSearch] = useState<string>("");
    const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>("connecting");
    const [isPaused, setIsPaused] = useState<boolean>(false);
    const isPausedRef = useRef<boolean>(isPaused);
    const bufferRef = useRef<FlatLogRecord[]>([]);
    const { data: serviceOptions, isLoading } = useDistinctServices();
    const dotColour = {
        connected: "success.main",
        connecting: "warning.main",
        error: "error.main",
    }[connectionStatus];

    const processBatch = useCallback((batch: FlatLogRecord[]) => {
        setLogs((prev) => {
            const combined = [...prev, ...batch];
            combined.sort((a: FlatLogRecord, b: FlatLogRecord) => b.timestamp - a.timestamp);
            return combined.slice(0, MAX_GLOBAL_LOGS);
        });
    }, []);

    const connect = useCallback(() => {
        if (wsRef.current?.readyState === WebSocket.OPEN) return;

        const ws = new WebSocket("ws://localhost:8091/ws/logs/tail");
        wsRef.current = ws;

        ws.onopen = () => {
            reconnectAttempts.current = 0; // reset backoff on successful connect
            setConnectionStatus("connected");
        };

        ws.onmessage = async (e) => {
            if (!e.data) return; // ignore empty messages
            const batch: FlatLogRecord[] = [];

            try {
                const buf = e.data instanceof Blob ? await e.data.arrayBuffer() : e.data;
                // decodeMulti parses the concatenated byte stream into individual objects
                for (const record of decodeMulti(buf, { extensionCodec })) {
                    batch.push(record as FlatLogRecord);
                }
            } catch {
                console.error("failed to parse websocket message", e.data);
                return;
            }

            if (!Array.isArray(batch) || batch.length === 0) return;

            if (isPausedRef.current) {
                bufferRef.current.push(...batch); // spread entire batch into buffer
                bufferRef.current = bufferRef.current.slice(-MAX_GLOBAL_LOGS);
                return;
            }

            processBatch(batch);
        };

        ws.onerror = (e) => {
            console.error("ws error", e);
        };

        ws.onclose = (e) => {
            // Close ghost websocket so that it does not trigger a reconnect
            if (wsRef.current !== ws) return;

            wsRef.current = null;
            if (e.code === WEBSOCKET_NORMAL_CLOSURE) return; // intentional close, don't reconnect

            const maxAttempts = 5;
            if (reconnectAttempts.current >= maxAttempts) {
                console.error("max reconnect attempts reached");
                setConnectionStatus("error");
                return;
            }

            setConnectionStatus("connecting");
            // Exponential backoff: 1s, 2s, 4s, 8s, 16s
            const delay = Math.min(1000 * 2 ** reconnectAttempts.current, 30_000);
            reconnectAttempts.current += 1;
            reconnectTimer.current = setTimeout(connect, delay);
        };
    }, []);

    useEffect(() => {
        connect();

        const cleanupConnection = () => {
            if (reconnectTimer.current) {
                clearTimeout(reconnectTimer.current);
            }
            if (wsRef.current) {
                wsRef.current.close(WEBSOCKET_NORMAL_CLOSURE, "navigating away");
            }
            setConnectionStatus("error");
        };

        return () => cleanupConnection();
    }, [connect]);

    useEffect(() => {
        isPausedRef.current = isPaused;
    }, [isPaused]);

    const handlePause = () => {
        setIsPaused(true);
    };

    const handleResume = () => {
        setIsPaused(false);

        if (bufferRef.current.length > 0) {
            processBatch(bufferRef.current);
            bufferRef.current = [];
        }
    };

    const handleReconnect = () => {
        setConnectionStatus("connecting");
        reconnectAttempts.current = 0;
        connect();
    };

    const handleServiceChange = useCallback(
        (event: SelectChangeEvent<string[]>) => {
            const value = event.target.value;
            setServices(typeof value === "string" ? value.split(",") : value);
        },
        [serviceOptions],
    );

    const handleSeverityChange = useCallback((event: SelectChangeEvent<LogType[]>) => {
        const value = event.target.value;
        setSeverities(typeof value === "string" ? (value.split(",") as LogType[]) : value);
    }, []);

    const filteredLogs = useMemo(() => {
        const result: FlatLogRecord[] = [];
        const lower = debouncedSearch.trim().toLowerCase();
        for (const log of logs) {
            if (result.length >= MAX_DISPLAY_LOGS) break;
            if (services.length > 0 && !services.includes(log.serviceName)) continue;
            if (severities.length > 0 && !severities.includes(parseSeverity(log.severityText))) continue;
            if (lower && !log.body.toLowerCase().includes(lower)) continue;
            result.push(log);
        }
        return result;
    }, [logs, services, severities, debouncedSearch]);

    return (
        <>
            <Box sx={{ ...card, width: "100%" }}>
                {/* Top Bar (Title and Pause button) */}
                <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 1 }}>
                    <Stack direction="row" sx={{ alignItems: "center" }} spacing={1}>
                        <Box sx={{ ...pulseSx, bgcolor: dotColour }} />
                        <Typography sx={{ ...sectionLabel, color: dotColour }}>Live Tail</Typography>
                    </Stack>
                    <Button
                        variant="outlined"
                        startIcon={isPaused ? <PlayArrowIcon fontSize="small" /> : <PauseIcon fontSize="small" />}
                        onClick={isPaused ? handleResume : handlePause}
                        disabled={connectionStatus !== "connected"}
                        sx={{
                            color: "text.disabled",
                            borderColor: "rgba(255,255,255,0.15)",
                            textTransform: "none",
                            fontSize: 13,
                            py: 0.5,
                            minWidth: 105,
                            "&.Mui-disabled": { borderColor: "rgba(255,255,255,0.05)" },
                        }}
                    >
                        {isPaused ? "Continue" : "Pause"}
                    </Button>
                </Box>

                {/* Filters Row */}
                <Stack direction="row" spacing={2} sx={{ mb: 3 }}>
                    <ServiceDropdown
                        services={services}
                        onChange={handleServiceChange}
                        isLoading={isLoading}
                        serviceOptions={serviceOptions?.services || []}
                    />

                    <SeverityDropdown severities={severities} onChange={handleSeverityChange} />
                    <SearchField onDebouncedChange={setDebouncedSearch} />
                </Stack>

                {/* WebSocket Error Alert Bar */}
                {connectionStatus === "error" && (
                    <ErrorBanner service="live tail server" handleReconnect={handleReconnect} />
                )}

                {/* Connecting alert bar */}
                {connectionStatus === "connecting" && (
                    <Box
                        sx={{
                            display: "flex",
                            alignItems: "center",
                            gap: 1,
                            width: "100%",
                            px: 2,
                            py: 1,
                            border: "1px solid rgba(20, 184, 166, 0.4)",
                            backgroundColor: "rgba(20, 184, 166, 0.08)",
                            borderRadius: "6px",
                            mb: 3,
                        }}
                    >
                        <CircularProgress size={14} thickness={5} sx={{ color: "#2dd4bf" }} />{" "}
                        <Typography variant="body2" sx={{ color: "#2dd4bf" }}>
                            Connecting to live tail server...
                        </Typography>
                    </Box>
                )}

                {/* Pause alert bar */}
                {isPaused && connectionStatus !== "error" && (
                    <Box
                        sx={{
                            display: "flex",
                            alignItems: "center",
                            gap: 1,
                            width: "100%",
                            px: 2,
                            py: 1,
                            border: "1px solid #78450a",
                            backgroundColor: "rgba(120, 69, 10, 0.15)",
                            borderRadius: "6px",
                            mb: 3,
                        }}
                    >
                        <PauseIcon sx={{ fontSize: 16, color: "warning.main" }} />
                        <Typography variant="body2" sx={{ color: "warning.main" }}>
                            Tail paused — new logs buffering
                        </Typography>
                    </Box>
                )}

                <TableContainer>
                    <Table>
                        <TableHead>
                            <TableRow>
                                {/* width set to 1% so that the columns will only span the length it occupies */}
                                <TableCell sx={{ color: "text.secondary", width: "1%" }}>TIME</TableCell>
                                <TableCell sx={{ color: "text.secondary", width: "1%" }}>SERVICE</TableCell>
                                <TableCell sx={{ color: "text.secondary", width: "1%" }}>SEVERITY</TableCell>
                                <TableCell sx={{ color: "text.secondary" }}>BODY</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {filteredLogs.length === 0 ? (
                                <TableRow>
                                    <TableCell colSpan={4} align="center" sx={{ color: "text.disabled", py: 4 }}>
                                        No logs match your filters
                                    </TableCell>
                                </TableRow>
                            ) : (
                                filteredLogs.slice(0, MAX_DISPLAY_LOGS).map((log) => {
                                    return (
                                        <TailLogRow
                                            key={`${log.spanId}-${log.timestamp}`}
                                            log={log}
                                            searchQuery={debouncedSearch}
                                        />
                                    );
                                })
                            )}
                        </TableBody>
                    </Table>
                </TableContainer>
            </Box>
        </>
    );
}
