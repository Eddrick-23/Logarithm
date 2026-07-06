import { Box, TableCell, TableRow, Typography } from "@mui/material";
import type { FlatLogRecord } from "../types/Log";
import { parseSeverity, severityStyles } from "../utils/severity";

interface TailLogRowProps {
    log: FlatLogRecord;
}

export default function TailLogRow({ log }: TailLogRowProps) {
    const severity = parseSeverity(log.severityText);
    return (
        <TableRow key={`${log.spanId}-${log.timestamp}`}>
            <TableCell sx={{ fontWeight: "bold", whiteSpace: "nowrap" }}>
                {new Date(log.timestamp).toLocaleString()}
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
            <TableCell sx={{ color: "text.disabled" }}>{log.body}</TableCell>
        </TableRow>
    );
}
