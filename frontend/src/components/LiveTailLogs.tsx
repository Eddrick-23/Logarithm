import {
    Box,
    Typography,
    Stack,
    Button,
    Select,
    MenuItem,
    TextField,
    type SelectChangeEvent,
    FormControl,
    InputLabel,
    InputAdornment,
    IconButton,
    CircularProgress,
} from "@mui/material";
import { useState, useRef, useEffect, useCallback } from "react";
import { card, pulseSx, sectionLabel } from "../theme/tokens";
import PauseIcon from "@mui/icons-material/Pause";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import SearchIcon from "@mui/icons-material/Search";
import ClearIcon from "@mui/icons-material/Clear";
import type { FlatLogRecord, LogType } from "../types/Log";
import { useDistinctServices } from "../hooks/useDistinctServices";
import TailLogRow, { columnWidths } from "./TailLogRow";
import ErrorBanner from "./ErrorBanner";
import { decodeMulti, ExtensionCodec } from "@msgpack/msgpack";

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

        // Convert to a JS Date and immediately return as an ISO string
        return new Date(seconds * 1000 + Math.floor(nanos / 1_000_000)).toISOString();
    },
});

type ConnectionStatus = "connecting" | "connected" | "error";

const LOG_TYPES: LogType[] = ["trace", "debug", "info", "warn", "error", "fatal"];
const MAX_GLOBAL_LOGS = 300;
const MAX_DISPLAY_LOGS = 15;
const WEBSOCKET_NORMAL_CLOSURE = 1000;
const DEBOUNCE_TIMEOUT = 300;

const SEVERITY_NORMALISE_MAP: Record<string, LogType> = {
    trace: "trace",
    debug: "debug",
    info: "info",
    warning: "warn", // logs coming in have severity text of WARNING
    warn: "warn", // by default OTEL uses "warn" instead of "warning" but we support both
    error: "error",
    fatal: "fatal",
};

const parseSeverity = (severityText: string | null | undefined): LogType => {
    // guard in case log has no severity text
    if (!severityText) return "info";
    // strip numbered variants: "DEBUG2" -> "debug", "WARN3" -> "warn"
    const base = severityText.replace(/\d+$/, "").toLowerCase();
    // fallback to info if we get an unsupported severity name
    return SEVERITY_NORMALISE_MAP[base] ?? "info";
};

export default function LiveTailLogs() {
    const wsRef = useRef<WebSocket | null>(null);
    const reconnectAttempts = useRef(0);
    const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
    const [logs, setLogs] = useState<FlatLogRecord[]>([]);
    const [severity, setSeverity] = useState<LogType | "all-severities">("all-severities");
    const [service, setService] = useState<string>("all-services");
    const [searchInput, setSearchInput] = useState<string>("");
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
            combined.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
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

            if (!Array.isArray(batch) || batch.length == 0) return;

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

    useEffect(() => {
        // set a delay until the user stops entering any search input
        const timer = setTimeout(() => setDebouncedSearch(searchInput), DEBOUNCE_TIMEOUT);
        return () => clearTimeout(timer);
    }, [searchInput]);

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

    const handleServiceChange = (event: SelectChangeEvent) => {
        setService(event.target.value);
    };

    const handleSeverityChange = (event: SelectChangeEvent) => {
        setSeverity(event.target.value as LogType | "all-severities");
    };

    const handleClear = () => {
        setSearchInput("");
    };

    const filteredLogs = logs.filter((log) => {
        if (service !== "all-services" && log.serviceName !== service) return false;
        if (severity !== "all-severities" && parseSeverity(log.severityText) !== severity) return false;

        if (debouncedSearch.trim()) {
            const lower = debouncedSearch.toLowerCase();
            const bodyMatch = log.body.toLowerCase().includes(lower);
            if (!bodyMatch) return false;
        }

        return true;
    });

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
                    <FormControl variant="outlined" sx={{ minWidth: 130 }}>
                        <InputLabel>Services</InputLabel>
                        <Select
                            value={service}
                            size="small"
                            label="Services"
                            onChange={handleServiceChange}
                            disabled={isLoading}
                            aria-label="services"
                        >
                            <MenuItem value="all-services">{isLoading ? "Loading..." : "All services"}</MenuItem>{" "}
                            {(serviceOptions?.services || []).map((serviceOption) => (
                                <MenuItem key={serviceOption} value={serviceOption}>
                                    {serviceOption}
                                </MenuItem>
                            ))}
                        </Select>
                    </FormControl>

                    <FormControl variant="outlined" sx={{ minWidth: 130 }}>
                        <InputLabel>Severity level</InputLabel>
                        <Select
                            value={severity}
                            size="small"
                            label="Severity level"
                            onChange={handleSeverityChange}
                            aria-label="severity level"
                        >
                            <MenuItem value="all-severities">All severities</MenuItem>
                            {LOG_TYPES.map((type) => (
                                <MenuItem key={type} value={type}>
                                    {type.toUpperCase()}
                                </MenuItem>
                            ))}
                        </Select>
                    </FormControl>

                    <TextField
                        value={searchInput}
                        onChange={(e) => setSearchInput(e.target.value)}
                        placeholder="Search body..."
                        size="small"
                        slotProps={{
                            input: {
                                startAdornment: (
                                    <InputAdornment position="start">
                                        <SearchIcon color="action" />
                                    </InputAdornment>
                                ),
                                endAdornment: searchInput && (
                                    <InputAdornment position="end">
                                        <IconButton onClick={handleClear} edge="end" size="small">
                                            <ClearIcon />
                                        </IconButton>
                                    </InputAdornment>
                                ),
                            },
                        }}
                    />
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

                {/* Table Headers */}
                <Box
                    sx={{
                        display: "flex",
                        alignItems: "center",
                        pb: 1.5,
                        borderBottom: "1px solid rgba(255,255,255,0.05)",
                        gap: 1,
                    }}
                >
                    <Typography
                        sx={{
                            width: columnWidths.time,
                            textAlign: "left",
                            fontSize: 13,
                            color: "text.secondary",
                            fontWeight: 600,
                        }}
                    >
                        TIME
                    </Typography>
                    <Typography
                        sx={{
                            width: columnWidths.service,
                            textAlign: "left",
                            fontSize: 13,
                            color: "text.secondary",
                            fontWeight: 600,
                        }}
                    >
                        SERVICE
                    </Typography>
                    <Typography
                        sx={{
                            width: columnWidths.severity,
                            textAlign: "left",
                            fontSize: 13,
                            color: "text.secondary",
                            fontWeight: 600,
                        }}
                    >
                        SEVERITY
                    </Typography>
                    <Typography
                        sx={{ flexGrow: 1, textAlign: "left", fontSize: 13, color: "text.secondary", fontWeight: 600 }}
                    >
                        BODY
                    </Typography>
                </Box>

                {/* Logs List */}
                <Box>
                    {filteredLogs
                        .slice(0, MAX_DISPLAY_LOGS)
                        .map(({ serviceName, timestamp, severityText, body }, index) => (
                            <TailLogRow
                                key={`${serviceName}-${timestamp}-${index}`}
                                time={new Date(timestamp).toLocaleString()}
                                service={serviceName}
                                severity={parseSeverity(severityText)}
                                message={body}
                            />
                        ))}

                    {filteredLogs.length === 0 && (
                        <Box sx={{ textAlign: "center", py: 3, color: "text.secondary" }}>
                            <Typography variant="body2">No logs match your filters</Typography>
                        </Box>
                    )}
                </Box>
            </Box>
        </>
    );
}
