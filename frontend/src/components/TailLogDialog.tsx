import { memo } from "react";
import {
    Dialog,
    DialogTitle,
    DialogContent,
    Table,
    TableBody,
    TableRow,
    TableCell,
    Typography,
    Divider,
    IconButton,
} from "@mui/material";
import type { FlatLogRecord } from "../types/Log";
import CloseIcon from "@mui/icons-material/Close";

type TailLogDialogProps = {
    onClose: () => void;
    log: FlatLogRecord | null;
};

function zipAttrs(keys: string[], values: string[]) {
    if (keys === undefined) return [];
    return keys.map((key, i) => ({ key, value: values[i] }));
}

export default memo(function TailLogDialog({ onClose, log }: TailLogDialogProps) {
    if (!log) return null;
    const open = log !== null;

    const logAttrs = zipAttrs(log.logAttrKeys, log.logAttrValues);
    const resAttrs = zipAttrs(log.resAttrKeys, log.resAttrValues);

    return (
        <Dialog open={open} onClose={onClose}>
            <DialogTitle sx={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
                Log Info
                <IconButton aria-label="close" onClick={onClose} size="small" sx={{ ml: 2 }}>
                    <CloseIcon fontSize="small" />
                </IconButton>
            </DialogTitle>

            <DialogContent dividers>
                <Table size="small">
                    <TableBody>
                        <TableRow>
                            <TableCell component="th">Timestamp</TableCell>
                            <TableCell>{new Date(log.timestamp / 1000).toLocaleString()}</TableCell>
                        </TableRow>
                        <TableRow>
                            <TableCell component="th">Trace ID</TableCell>
                            <TableCell sx={{ fontFamily: "monospace" }}>{log.traceId}</TableCell>
                        </TableRow>
                        <TableRow>
                            <TableCell component="th">Span ID</TableCell>
                            <TableCell sx={{ fontFamily: "monospace" }}>{log.spanId}</TableCell>
                        </TableRow>
                        <TableRow>
                            <TableCell component="th">Severity</TableCell>
                            <TableCell>
                                {log.severityText} ({log.severityNumber})
                            </TableCell>
                        </TableRow>
                        <TableRow>
                            <TableCell component="th">Service Name</TableCell>
                            <TableCell>{log.serviceName}</TableCell>
                        </TableRow>
                        <TableRow>
                            <TableCell component="th">Scope Name</TableCell>
                            <TableCell>{log.scopeName}</TableCell>
                        </TableRow>
                        <TableRow>
                            <TableCell component="th">Scope Version</TableCell>
                            <TableCell>{log.scopeVersion}</TableCell>
                        </TableRow>
                        <TableRow>
                            <TableCell component="th">Body</TableCell>
                            <TableCell sx={{ whiteSpace: "pre-wrap", wordBreak: "break-word" }}>{log.body}</TableCell>
                        </TableRow>
                    </TableBody>
                </Table>

                <Divider sx={{ my: 2 }} />

                <Typography variant="subtitle2" gutterBottom>
                    Log Attributes
                </Typography>
                <Table size="small">
                    <TableBody>
                        {logAttrs.length === 0 && (
                            <TableRow>
                                <TableCell colSpan={2}>
                                    <Typography variant="body2" color="text.secondary">
                                        None
                                    </Typography>
                                </TableCell>
                            </TableRow>
                        )}
                        {logAttrs.map(({ key, value }) => (
                            <TableRow key={key}>
                                <TableCell component="th">{key}</TableCell>
                                <TableCell sx={{ wordBreak: "break-word" }}>{value}</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>

                <Divider sx={{ my: 2 }} />

                <Typography variant="subtitle2" gutterBottom>
                    Resource Attributes
                </Typography>
                <Table size="small">
                    <TableBody>
                        {resAttrs.length === 0 && (
                            <TableRow>
                                <TableCell colSpan={2}>
                                    <Typography variant="body2" color="text.secondary">
                                        None
                                    </Typography>
                                </TableCell>
                            </TableRow>
                        )}
                        {resAttrs.map(({ key, value }) => (
                            <TableRow key={key}>
                                <TableCell component="th">{key}</TableCell>
                                <TableCell sx={{ wordBreak: "break-word" }}>{value}</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </DialogContent>
        </Dialog>
    );
});
