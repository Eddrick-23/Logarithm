import {
    Box,
    Typography,
    Stack,
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
import { card } from "../theme/tokens";
import PauseIcon from "@mui/icons-material/Pause";
import type { FlatLogRecord, LogType } from "../types/Log";
import { useDistinctServices } from "../hooks/useDistinctServices";
import ErrorBanner from "./ErrorBanner";
import { parseSeverity } from "../utils/severity";
import TailLogRow from "./TailLogRow";
import SearchField from "./SearchField";
import SeverityDropdown from "./SeverityDropdown";
import ServiceDropdown from "./ServiceDropdown";
import LiveTailLogsHeader from "./LiveTailLogsHeader";
import type { ConnectionStatus } from "../types/Connection";

const MAX_DISPLAY_LOGS = 15;

export default function LiveTailLogs() {
    const workerRef = useRef<Worker | null>(null);
    const [displayLogs, setDisplayLogs] = useState<FlatLogRecord[]>([]);
    const [severities, setSeverities] = useState<LogType[]>([]); // empty indicates all severities selected
    const [services, setServices] = useState<string[]>([]); // empty indicates all services selected
    const [debouncedSearch, setDebouncedSearch] = useState<string>("");
    const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>("connecting");
    const [isPaused, setIsPaused] = useState<boolean>(false);
    const { data: serviceOptions, isLoading } = useDistinctServices();

    useEffect(() => {
        // Instantiate the worker using Vite's standard pattern
        workerRef.current = new Worker(
            new URL("../workers/logWorker.ts", import.meta.url),
            { type: "module" }, // Required to allow imports inside the worker
        );

        workerRef.current.onmessage = (e) => {
            if (e.data.type === "LOG_UPDATE") {
                setDisplayLogs(e.data.payload);
            }
            // Listen for status changes from the worker
            if (e.data.type === "STATUS") {
                setConnectionStatus(e.data.payload);
            }
        };

        return () => {
            // Tell the worker to close the WS cleanly before terminating
            workerRef.current?.postMessage({ type: "CLEANUP" });
            workerRef.current?.terminate();
        };
    }, []);

    const handlePause = useCallback(() => {
        setIsPaused(true);
        workerRef.current?.postMessage({ type: "PAUSE" });
    }, []);

    const handleResume = useCallback(() => {
        setIsPaused(false);
        workerRef.current?.postMessage({ type: "RESUME" });
    }, []);

    const handleReconnect = () => {
        setConnectionStatus("connecting");
        workerRef.current?.postMessage({ type: "RECONNECT" });
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
        for (const log of displayLogs) {
            if (result.length >= MAX_DISPLAY_LOGS) break;
            if (services.length > 0 && !services.includes(log.serviceName)) continue;
            if (severities.length > 0 && !severities.includes(parseSeverity(log.severityText))) continue;
            if (lower && !log.body.toLowerCase().includes(lower)) continue;
            result.push(log);
        }
        return result;
    }, [displayLogs, services, severities, debouncedSearch]);

    return (
        <>
            <Box sx={{ ...card, width: "100%" }}>
                {/* Top Bar (Title and Pause button) */}
                <LiveTailLogsHeader
                    isPaused={isPaused}
                    handleResume={handleResume}
                    handlePause={handlePause}
                    connectionStatus={connectionStatus}
                />

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
                            {displayLogs.length === 0 ? (
                                // No logs have arrived yet.
                                <TableRow>
                                    <TableCell colSpan={4} align="center" sx={{ color: "text.disabled", py: 4 }}>
                                        Waiting for the first log to arrive...
                                    </TableCell>
                                </TableRow>
                            ) : filteredLogs.length === 0 ? (
                                // No logs match the filters of service, severity or body filters
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
