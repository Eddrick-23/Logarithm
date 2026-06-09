import { Box, Typography } from "@mui/material";
import { logRowSx } from "../theme/tokens";
import type { LogType } from "../types/Log";

interface TailLogProps {
    time: string;
    service: string;
    severity: LogType;
    message: string;
}

// Centralized fixed widths for perfect alignment
export const columnWidths = {
    time: 160,
    service: 110,
    severity: 110,
};

// Styling maps for the severity badges
const severityStyles: Record<LogType, { bg: string; text: string }> = {
    trace: { bg: "rgba(189, 189, 189, 0.15)", text: "#bdbdbd" }, // Grey
    debug: { bg: "rgba(100, 181, 246, 0.15)", text: "#64b5f6" }, // Blue
    info: { bg: "rgba(102, 187, 106, 0.15)", text: "#66bb6a" }, // Green
    warn: { bg: "rgba(255, 167, 38, 0.15)", text: "#ffa726" }, // Orange
    error: { bg: "rgba(239, 83, 80, 0.15)", text: "#ef5350" }, // Red
    fatal: { bg: "rgba(171, 71, 188, 0.15)", text: "#ab47bc" }, // Purple
};

export default function TailLogRow({ time, service, severity, message }: TailLogProps) {
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
