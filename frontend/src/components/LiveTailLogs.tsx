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
} from "@mui/material";
import { useState, useRef, useEffect, useCallback } from "react";
import { card, pulseSx, sectionLabel } from "../theme/tokens";
import PauseIcon from "@mui/icons-material/Pause";
import ErrorIcon from "@mui/icons-material/Error";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import SearchIcon from "@mui/icons-material/Search";
import ClearIcon from "@mui/icons-material/Clear";
import type { FlatLogEntry, LogIngestRequest, LogType } from "../types/Log";
import { useDistinctServices } from "../hooks/useDistinctServices";
import TailLogRow, { columnWidths } from "./TailLogRow";

const LOG_TYPES: LogType[] = ["debug", "info", "warning", "error"];
const MAX_GLOBAL_LOGS = 300;
const MAX_DISPLAY_LOGS = 15;

const parseSeverity = (severityText: string): LogType => {
    const lower = severityText.toLowerCase();
    return lower as LogType;
};

export default function LiveTailLogs() {
    const wsRef = useRef<WebSocket | null>(null);
    const reconnectAttempts = useRef(0);
    const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
    const [logs, setLogs] = useState<FlatLogEntry[]>([]);
    const [severity, setSeverity] = useState<LogType | "all-severities">("all-severities");
    const [service, setService] = useState<string>("all-services");
    const [searchInput, setSearchInput] = useState<string>("");
    const [debouncedSearch, setDebouncedSearch] = useState<string>("");
    const [hasConnectionError, setHasConnectionError] = useState<boolean>(false);
    const [isPaused, setIsPaused] = useState<boolean>(false);
    const isPausedRef = useRef<boolean>(isPaused);
    const bufferRef = useRef<LogIngestRequest[]>([]);
    const isInitializing = useRef(true); // true until first successful connect or intentional close
    const { data: serviceOptions, isLoading } = useDistinctServices();

    const processBatch = useCallback((batch: LogIngestRequest[]) => {
        const newEntries: FlatLogEntry[] = batch.flatMap(({ serviceName, records }) =>
            records.map((log) => ({ serviceName, log })),
        );

        setLogs((prev) => {
            const combined = [...prev, ...newEntries];
            combined.sort((a, b) => new Date(b.log.timestamp).getTime() - new Date(a.log.timestamp).getTime());
            return combined.slice(0, MAX_GLOBAL_LOGS);
        });
    }, []);

    const connect = useCallback(() => {
        if (wsRef.current?.readyState === WebSocket.OPEN) return;

        const ws = new WebSocket("ws://localhost:8091/ws/logs/tail");
        wsRef.current = ws;

        ws.onopen = () => {
            reconnectAttempts.current = 0; // reset backoff on successful connect
            isInitializing.current = false;
            setHasConnectionError(false);
        };

        ws.onmessage = (e) => {
            const batch: LogIngestRequest[] = JSON.parse(e.data);

            if (isPausedRef.current) {
                bufferRef.current.push(...batch); // spread entire batch into buffer
                return;
            }

            processBatch(batch);
        };

        ws.onerror = (e) => {
            console.error("ws error", e);
            if (isInitializing.current) return; // suppress mount-time error
            setHasConnectionError(true);
            handlePause();
        };

        ws.onclose = (e) => {
            // Close ghost websocket so that it does not trigger a reconnect
            if (wsRef.current !== ws) return;

            wsRef.current = null;
            isInitializing.current = false;
            if (e.code === 1000) return; // intentional close, don't reconnect

            const maxAttempts = 5;
            if (reconnectAttempts.current >= maxAttempts) {
                console.error("max reconnect attempts reached");
                setHasConnectionError(true);
                return;
            }

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
                wsRef.current.close(1000, "navigating away");
            }
        };

        return () => cleanupConnection();
    }, [connect]);

    useEffect(() => {
        isPausedRef.current = isPaused;
    }, [isPaused]);

    useEffect(() => {
        // set a delay until the user stops entering any search input
        const timer = setTimeout(() => setDebouncedSearch(searchInput), 300);
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

    const filteredLogs = logs.filter(({ log, serviceName }) => {
        if (service !== "all-services" && serviceName !== service) return false;
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
                        <Box sx={{ ...pulseSx, bgcolor: hasConnectionError ? "error.main" : "success.main" }} />
                        <Typography sx={{ ...sectionLabel }}>Live Tail</Typography>
                    </Stack>
                    <Button
                        variant="outlined"
                        startIcon={isPaused ? <PlayArrowIcon fontSize="small" /> : <PauseIcon fontSize="small" />}
                        onClick={isPaused ? handleResume : handlePause}
                        disabled={hasConnectionError}
                        sx={{
                            color: "#9e9e9e",
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
                        >
                            <MenuItem value="all-services">{isLoading ? "Loading..." : "All services"}</MenuItem>{" "}
                            {serviceOptions?.services.map((serviceOption) => (
                                <MenuItem key={serviceOption} value={serviceOption}>
                                    {serviceOption}
                                </MenuItem>
                            ))}
                        </Select>
                    </FormControl>

                    <FormControl variant="outlined" sx={{ minWidth: 130 }}>
                        <InputLabel>Severity level</InputLabel>
                        <Select value={severity} size="small" label="Severity level" onChange={handleSeverityChange}>
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
                {hasConnectionError && (
                    <Box
                        sx={{
                            display: "flex",
                            alignItems: "center",
                            gap: 1,
                            width: "100%",
                            px: 2,
                            py: 1,
                            border: "1px solid #7f1d1d",
                            backgroundColor: "rgba(127, 29, 29, 0.15)",
                            borderRadius: "6px",
                            mb: 3,
                        }}
                    >
                        <ErrorIcon sx={{ fontSize: 16, color: "#ef4444" }} />
                        <Typography variant="body2" sx={{ color: "#ef4444" }}>
                            Connection lost. Failed to connect to the live tail server.{" "}
                        </Typography>
                        <Button
                            size="small"
                            sx={{
                                ml: "auto",
                                color: "#ef4444",
                                borderColor: "#ef4444",
                                textTransform: "none",
                                fontSize: 12,
                                "&:hover": { borderColor: "#ef4444", backgroundColor: "rgba(239, 68, 68, 0.08)" },
                            }}
                            variant="outlined"
                            onClick={handleReconnect}
                        >
                            Retry
                        </Button>
                    </Box>
                )}

                {/* Pause alert bar */}
                {isPaused && !hasConnectionError && (
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
                        <PauseIcon sx={{ fontSize: 16, color: "#c97316" }} />
                        <Typography variant="body2" sx={{ color: "#c97316" }}>
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
                            color: "#6e7681",
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
                            color: "#6e7681",
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
                            color: "#6e7681",
                            fontWeight: 600,
                        }}
                    >
                        SEVERITY
                    </Typography>
                    <Typography
                        sx={{ flexGrow: 1, textAlign: "left", fontSize: 13, color: "#6e7681", fontWeight: 600 }}
                    >
                        BODY
                    </Typography>
                </Box>

                {/* Logs List */}
                <Box>
                    {filteredLogs.slice(0, MAX_DISPLAY_LOGS).map(({ log, serviceName }, index) => (
                        <TailLogRow
                            key={`${serviceName}-${log.timestamp}-${index}`}
                            time={new Date(log.timestamp).toLocaleString()}
                            service={serviceName}
                            severity={parseSeverity(log.severityText)}
                            message={log.body}
                        />
                    ))}

                    {filteredLogs.length === 0 && (
                        <Box sx={{ textAlign: "center", py: 3, color: "#6e7681" }}>
                            <Typography variant="body2">No logs match your filters</Typography>
                        </Box>
                    )}
                </Box>
            </Box>
        </>
    );
}
