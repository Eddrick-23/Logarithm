import { useMemo } from "react";
import {
    Box,
    Typography,
    Paper,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Alert,
    Skeleton,
} from "@mui/material";
import type { ConfigItem } from "../types/Config";
import { useConfig } from "../hooks/useConfig";

const CONFIG_ITEMS: Omit<ConfigItem, "value">[] = [
    {
        key: "liveTailRefreshInterval",
        label: "Live tail refresh interval",
        description: "How often the live tail view polls for new logs, in milliseconds.",
    },
    {
        key: "liveTailMaxBatch",
        label: "Live tail max batch",
        description: "Maximum number of logs delivered per live tail update.",
    },
    {
        key: "natsStreamMaxAge",
        label: "NATS stream max age",
        description: "How long ingested logs in NATS stream are kept before being deleted.",
    },
    {
        key: "natsDLQMaxAge",
        label: "NATS dead-letter queue max age",
        description: "How long failed/undeliverable logs are kept before being deleted.",
    },
    {
        key: "natsMaxDeliver",
        label: "NATS max delivery attempts",
        description: "Maximum number of times NATS will attempt to redeliver a message before it's dead-lettered.",
    },
    {
        key: "natsBackoff",
        label: "NATS redelivery backoff",
        description: "Delay schedule between successive redelivery attempts for failed NATS messages.",
    },
    {
        key: "natsLogStreamMaxBytes",
        label: "NATS log stream max bytes",
        description: "Maximum disk space allocated to the log stream before oldest data is evicted.",
    },
    {
        key: "natsConsumerMaxAckPending",
        label: "NATS consumer max pending acknowledgements",
        description: "Maximum number of unacknowledged messages a consumer may have in flight.",
    },
    {
        key: "natsDLQMaxBytes",
        label: "NATS dead-letter queue max bytes",
        description: "Maximum disk space allocated to the dead-letter queue.",
    },
    {
        key: "workerLogLevel",
        label: "Worker log level",
        description: "Verbosity of logs emitted by background workers.",
    },
    {
        key: "workerMaxBatch",
        label: "Worker max batch",
        description: "Maximum number of records a worker processes in a single batch.",
    },
    {
        key: "workerBackoff",
        label: "Worker retry backoff",
        description: "Delay schedule between successive retry attempts for failed worker batches.",
    },
    {
        key: "workerRowsPerBatch",
        label: "Worker rows per batch",
        description: "Number of rows written to storage per worker batch.",
    },
];

export default function Search() {
    const { data, isLoading, isError } = useConfig();

    const rows: ConfigItem[] = useMemo(
        () =>
            CONFIG_ITEMS.map((item) => ({
                ...item,
                value: data?.[item.key] ?? "—",
            })),
        [data],
    );

    return (
        <Box sx={{ mx: "auto", px: 3, py: 2 }}>
            {/* loading skeleton */}
            {isLoading && (
                <Box>
                    <Skeleton variant="text" width={220} height={32} />
                    <Skeleton variant="text" width={420} sx={{ mb: 2 }} />
                    <Skeleton variant="rectangular" height={42} sx={{ mb: 1, borderRadius: 1 }} />
                    {Array.from({ length: 8 }).map((_, i) => (
                        <Skeleton key={i} variant="text" height={32} />
                    ))}
                </Box>
            )}

            {/* error alert */}
            {isError && <Alert severity="error">Failed to load configurations.</Alert>}

            {/* config table */}
            {!isLoading && !isError && (
                <Box>
                    <Typography variant="h5">System configuration</Typography>
                    <Typography variant="body2" sx={{ color: "text.secondary", mt: 0.5, mb: 3 }}>
                        Current operational settings for this deployment. To update the settings, update your
                        environment variables and restart Logarithm.
                    </Typography>

                    <TableContainer
                        component={Paper}
                        variant="outlined"
                        sx={{ border: "2px solid", borderColor: "divider" }}
                    >
                        <Table>
                            <TableHead>
                                <TableRow sx={{ "& th": { bgcolor: "background.default", fontWeight: 600 } }}>
                                    <TableCell>Setting</TableCell>
                                    <TableCell>Description</TableCell>
                                    <TableCell>Value</TableCell>
                                </TableRow>
                            </TableHead>

                            <TableBody>
                                {rows.map((row) => (
                                    <TableRow key={row.key} hover>
                                        <TableCell>{row.label}</TableCell>
                                        <TableCell sx={{ color: "text.secondary" }}>{row.description}</TableCell>
                                        <TableCell>
                                            <Typography component="span" sx={{ fontFamily: "monospace", fontSize: 13 }}>
                                                {row.value}
                                            </Typography>
                                        </TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    </TableContainer>
                </Box>
            )}
        </Box>
    );
}
