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
} from "@mui/material";
import type { LogType } from "../types/LogType";
import { useState, useRef, useEffect, useCallback } from "react";
import { card, logRowSx, pulseSx, sectionLabel } from "../theme/tokens";
import PauseIcon from "@mui/icons-material/Pause";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";

const LOG_TYPES: LogType[] = ["debug", "info", "warning", "error"];

interface TailLogProps {
    time: string;
    service: string;
    severity: LogType;
    message: string;
}

// TODO: replace dummy data with live data
const logData: TailLogProps[] = [
    {
        time: "14:18:00.422",
        service: "payments",
        severity: "error",
        message: "health check passed, all dependencies reachable",
    },
    {
        time: "14:17:59.607",
        service: "notifier",
        severity: "error",
        message: "stock low for item_id=4491, qty=3 remaining",
    },
    {
        time: "14:17:58.804",
        service: "payments",
        severity: "info",
        message: "stock low for item_id=4491, qty=3 remaining",
    },
    {
        time: "14:17:58.004",
        service: "auth-svc",
        severity: "info",
        message: "smtp timeout after 5000ms, retrying (2/3)",
    },
    {
        time: "14:17:57.201",
        service: "worker",
        severity: "error",
        message: "charge processed txn_id=TXN-08441 — $48.00",
    },
    { time: "14:17:56.393", service: "auth-svc", severity: "warning", message: "token validated for user_id=8821" },
    {
        time: "14:17:55.579",
        service: "worker",
        severity: "info",
        message: "cache miss for key user:9912, fetching from db",
    },
    {
        time: "14:17:54.770",
        service: "db-proxy",
        severity: "info",
        message: "health check passed, all dependencies reachable",
    },
    {
        time: "14:17:53.969",
        service: "inventory",
        severity: "warning",
        message: "rate limit exceeded for client_id=772",
    },
    {
        time: "14:17:53.144",
        service: "payments",
        severity: "debug",
        message: "bug in payment processes",
    },
];

// Styling maps for the severity badges
const severityStyles: Record<LogType, { bg: string; text: string }> = {
    debug: { bg: "rgba(100, 181, 246, 0.15)", text: "#64b5f6" }, // Blue
    info: { bg: "rgba(102, 187, 106, 0.15)", text: "#66bb6a" }, // Green
    warning: { bg: "rgba(255, 167, 38, 0.15)", text: "#ffa726" }, // Orange
    error: { bg: "rgba(239, 83, 80, 0.15)", text: "#ef5350" }, // Red
};

// Centralized fixed widths for perfect alignment
const columnWidths = {
    time: 130,
    service: 110,
    severity: 110,
};

function TailLogRow({ time, service, severity, message }: TailLogProps) {
    return (
        <Box sx={{ ...logRowSx, display: "flex", alignItems: "center", py: 1.5, fontSize: 14 }}>
            {/* TIME */}
            <Typography
                sx={{ width: columnWidths.time, textAlign: "left", flexShrink: 0, fontSize: 13, fontWeight: "bold" }}
            >
                {time}
            </Typography>

            {/* SERVICE TAG */}
            <Box sx={{ width: columnWidths.service, flexShrink: 0, textAlign: "left" }}>
                <Box
                    sx={{
                        display: "inline-block",
                        border: "1px solid rgba(255,255,255,0.15)",
                        borderRadius: 1,
                        px: 1,
                        py: 0.25,
                    }}
                >
                    <Typography sx={{ fontSize: 12, color: "#9e9e9e" }}>{service}</Typography>
                </Box>
            </Box>

            {/* SEVERITY TAG */}
            <Box sx={{ width: columnWidths.severity, flexShrink: 0, textAlign: "left" }}>
                <Box
                    sx={{
                        display: "inline-block",
                        backgroundColor: severityStyles[severity].bg,
                        borderRadius: 1,
                        px: 1,
                        py: 0.25,
                    }}
                >
                    <Typography
                        sx={{
                            fontSize: 11,
                            fontWeight: "bold",
                            color: severityStyles[severity].text,
                        }}
                    >
                        {severity.toUpperCase()}
                    </Typography>
                </Box>
            </Box>

            {/* BODY */}
            <Typography
                sx={{
                    flexGrow: 1,
                    textAlign: "left",
                    fontSize: 13,
                    color: "#9e9e9e",
                    overflowWrap: "break-word",
                }}
            >
                {message}
            </Typography>
        </Box>
    );
}

export default function LiveTailLogs() {
    const wsRef = useRef<WebSocket | null>(null);
    const reconnectAttempts = useRef(0);
    const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

    const connect = useCallback(() => {
        if (wsRef.current?.readyState === WebSocket.OPEN) return;

        const ws = new WebSocket("ws://localhost:8091/ws/logs/tail");
        wsRef.current = ws;

        ws.onopen = () => {
            console.log("connected");
            reconnectAttempts.current = 0; // reset backoff on successful connect
        };

        ws.onmessage = (e) => {
            console.log("data", e.data);
            // setLogs((prev) => [...prev, e.data]);
        };

        ws.onerror = (e) => console.error("ws error", e);

        ws.onclose = (e) => {
            wsRef.current = null;

            if (e.code === 1000) return; // intentional close, don't reconnect

            const maxAttempts = 5;
            if (reconnectAttempts.current >= maxAttempts) {
                console.error("max reconnect attempts reached");
                return;
            }

            // Exponential backoff: 1s, 2s, 4s, 8s, 16s
            const delay = Math.min(1000 * 2 ** reconnectAttempts.current, 30_000);
            reconnectAttempts.current += 1;

            console.log(`reconnecting in ${delay}ms (attempt ${reconnectAttempts.current})`);
            reconnectTimer.current = setTimeout(connect, delay);
        };
    }, []);

    useEffect(() => {
        connect();

        return () => {
            // Cancel any pending reconnect
            if (reconnectTimer.current) clearTimeout(reconnectTimer.current);
            wsRef.current?.close(1000, "component unmounted");
        };
    }, [connect]);

    // TODO: connect to backend API and update logs in real time
    const [severity, setSeverity] = useState<LogType | "all-severities">("all-severities");
    const [service, setService] = useState<string>("all-services");
    const [isPaused, setIsPaused] = useState<boolean>(false);
    const handlePause = () => {
        // TODO: add fetching logic
        setIsPaused(true);
    };

    const handleResume = () => {
        // TODO: add fetching logic
        setIsPaused(false);
    };

    const handleServiceChange = (event: SelectChangeEvent) => {
        setService(event.target.value);
    };

    const handleSeverityChange = (event: SelectChangeEvent) => {
        setSeverity(event.target.value as LogType | "all-severities");
    };

    return (
        <>
            <Box sx={{ ...card, width: "100%" }}>
                {/* Top Bar (Title and Pause button) */}
                <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 1 }}>
                    <Stack direction="row" sx={{ alignItems: "center" }} spacing={1}>
                        <Box sx={{ ...pulseSx, color: "success.main" }} />
                        <Typography sx={{ ...sectionLabel }}>Live Tail</Typography>
                    </Stack>
                    <Button
                        variant="outlined"
                        startIcon={isPaused ? <PlayArrowIcon fontSize="small" /> : <PauseIcon fontSize="small" />}
                        onClick={isPaused ? handleResume : handlePause}
                        sx={{
                            color: "#9e9e9e",
                            borderColor: "rgba(255,255,255,0.15)",
                            textTransform: "none",
                            fontSize: 13,
                            py: 0.5,
                            minWidth: 105,
                        }}
                    >
                        {isPaused ? "Continue" : "Pause"}
                    </Button>
                </Box>

                {/* Filters Row */}
                <Stack direction="row" spacing={2} sx={{ mb: 3 }}>
                    <FormControl variant="outlined" sx={{ minWidth: 130 }}>
                        <InputLabel>Services</InputLabel>
                        <Select value={service} size="small" label="Services" onChange={handleServiceChange}>
                            <MenuItem value="all-services">All services</MenuItem>
                            {/* TODO: update service filters */}
                            <MenuItem value="service-1">Service 1</MenuItem>
                            <MenuItem value="service-2">Service 2</MenuItem>
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

                    <TextField placeholder="Search body..." size="small" />
                </Stack>

                {/* Pause alert bar */}
                {isPaused && (
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
                    {logData.map((log, index) => (
                        <TailLogRow
                            key={`${log.time}-${index}`}
                            time={log.time}
                            service={log.service}
                            severity={log.severity}
                            message={log.message}
                        />
                    ))}
                </Box>
            </Box>
        </>
    );
}
