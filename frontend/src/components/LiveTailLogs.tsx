import { Box, Typography, Stack, Button, Select, MenuItem, TextField } from "@mui/material";
import type { LogType } from "../types/LogType";
import { card, logRowSx, pulseSx, sectionLabel } from "../theme/tokens";
import PauseIcon from "@mui/icons-material/Pause";

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
        severity: "info",
        message: "stock low for item_id=4491, qty=3 remaining",
    },
];

// Styling maps for the severity badges
const severityStyles: Record<LogType, { bg: string; text: string }> = {
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
    // TODO: connect to backend API and update logs in real time

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
                        startIcon={<PauseIcon fontSize="small" />}
                        sx={{
                            color: "#9e9e9e",
                            borderColor: "rgba(255,255,255,0.15)",
                            textTransform: "none",
                            fontSize: 13,
                            py: 0.5,
                        }}
                    >
                        Pause
                    </Button>
                </Box>

                {/* Filters Row */}
                {/* TODO: add more filters */}
                <Stack direction="row" spacing={2} sx={{ mb: 3 }}>
                    <Select value="all-services" size="small">
                        <MenuItem value="all-services">All services</MenuItem>
                    </Select>

                    <Select value="all-severities" size="small">
                        <MenuItem value="all-severities">All severities</MenuItem>
                    </Select>

                    <TextField placeholder="Search body..." size="small" />
                </Stack>

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
                            fontSize: 11,
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
                            fontSize: 11,
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
                            fontSize: 11,
                            color: "#6e7681",
                            fontWeight: 600,
                        }}
                    >
                        SEVERITY
                    </Typography>
                    <Typography
                        sx={{ flexGrow: 1, textAlign: "left", fontSize: 11, color: "#6e7681", fontWeight: 600 }}
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
