import { Box, IconButton, TableCell, TableRow, Typography } from "@mui/material";
import type { FlatLogRecord } from "../types/Log";
import { parseSeverity, severityStyles } from "../utils/severity";
import { memo } from "react";
import InfoIcon from "@mui/icons-material/Info";

interface HighlightedBodyProps {
    text: string;
    query?: string;
}
interface TailLogRowProps {
    log: FlatLogRecord;
    searchQuery?: string;
    onInfoClick: (log: FlatLogRecord) => void;
}

// prevents special characters in search input (".", "*", "(") from breaking regex
function escapeRegExp(value: string): string {
    return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function HighlightedBody({ text, query }: HighlightedBodyProps) {
    // if there is no query or query is just spaces, return text
    if (!query || !query.trim()) return <>{text}</>;

    const escaped = escapeRegExp(query.trim()); // sanitise the query
    const parts = text.split(new RegExp(`(${escaped})`, "gi")); // g match every occurence, i makes it case-insensitive

    return (
        <>
            {parts.map((part, i) =>
                part.toLowerCase() === query.trim().toLowerCase() ? (
                    <Box
                        key={i}
                        component="mark"
                        sx={{
                            backgroundColor: "rgba(250, 204, 21, 0.35)",
                            color: "inherit",
                            borderRadius: "3px",
                            px: "2px",
                        }}
                    >
                        {part}
                    </Box>
                ) : (
                    <span key={i}>{part}</span>
                ),
            )}
        </>
    );
}

export default memo(function TailLogRow({ log, searchQuery, onInfoClick }: TailLogRowProps) {
    const severity = parseSeverity(log.severityText);
    return (
        <TableRow>
            <TableCell sx={{ fontWeight: "bold", whiteSpace: "nowrap" }}>
                {/* date accept timing in milliseconds and the timestamp is in microseconds */}
                {new Date(log.timestamp / 1_000).toLocaleString()}
            </TableCell>
            <TableCell sx={{ color: "text.disabled", whiteSpace: "nowrap" }}>{log.serviceName}</TableCell>
            <TableCell>
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
            </TableCell>
            <TableCell sx={{ color: "text.disabled" }}>
                <HighlightedBody text={log.body} query={searchQuery} />
            </TableCell>
            <TableCell>
                <IconButton onClick={() => onInfoClick(log)}>
                    <InfoIcon />
                </IconButton>
            </TableCell>
        </TableRow>
    );
});
