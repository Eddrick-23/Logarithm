import { LineChart } from "@mui/x-charts/LineChart";
import { useMemo } from "react";
import { useMetrics } from "../hooks/useMetrics";
import { Alert, Box, Skeleton, Typography } from "@mui/material";
import { card, sectionLabel } from "../theme/tokens";
import ErrorBanner from "./ErrorBanner";

const CHART_HEIGHT = 400;

// automatically generates the colour based on golden angle formula
const generateColour = (index: number) => `hsl(${(index * 137.5) % 360}, 70%, 50%)`;

export default function IngestionGraph() {
    const { data, isLoading, isError, refetch } = useMetrics();

    const services = useMemo(() => Object.keys(data ?? {}), [data]);

    const timestamps = useMemo(() => {
        if (!data || services.length === 0) return [];
        // with the backfill of the timestamps performed in clickhouse db
        // it guarantees that all timestamps will be present and we can
        // just use the first service's timestamps as the x-axis spine
        return (data[services[0]] ?? []).map((metric) => metric.timestamp);
    }, [data, services]);

    const series = useMemo(() => {
        if (!data) return [];
        return services.map((service, index) => ({
            label: service,
            data: (data[service] ?? []).map((metric) => metric.logsCount),
            color: generateColour(index),
        }));
    }, [data, services]);

    const isEmpty = !isLoading && !isError && services.length === 0;

    return (
        <Box sx={{ ...card }}>
            <Typography sx={sectionLabel}>Ingestion Throughput - Last 60s</Typography>

            {/* rectangular skeleton box to signify loading of graph */}
            {isLoading && (
                <Skeleton variant="rectangular" width="100%" height={CHART_HEIGHT} sx={{ borderRadius: 2 }} />
            )}

            {/* error banner */}
            {isError && <ErrorBanner service="server" handleReconnect={refetch} />}

            {/* no logs received from api */}
            {isEmpty && <Alert severity="info">No logs received in the last 60 seconds</Alert>}

            {!isLoading && !isEmpty && (
                <LineChart
                    xAxis={[
                        {
                            data: timestamps,
                            scaleType: "point",
                            valueFormatter: (v) => new Date(v).toLocaleTimeString(),
                            label: "Time",
                        },
                    ]}
                    yAxis={[{ min: 0, label: "Logs / sec" }]} // set min to 0 so that y starts from 0
                    series={series}
                    height={CHART_HEIGHT}
                />
            )}
        </Box>
    );
}
