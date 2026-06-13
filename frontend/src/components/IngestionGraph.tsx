import { LineChart } from "@mui/x-charts/LineChart";
import { useMemo } from "react";
import { useIngestionMetrics } from "../hooks/useMetrics";
import { Alert, Box, Skeleton, Typography } from "@mui/material";
import { card, sectionLabel } from "../theme/tokens";
import ErrorBanner from "./ErrorBanner";

const CHART_HEIGHT = 400;

// automatically generates the colour based on golden angle formula
const generateColour = (index: number) => `hsl(${(index * 137.5) % 360}, 70%, 50%)`;

export default function IngestionGraph() {
    const { data, isLoading, isError, refetch } = useIngestionMetrics();
    const services = useMemo(() => Object.keys(data?.metrics ?? {}), [data]);

    const series = useMemo(() => {
        if (!data) return [];
        return services.map((service, index) => ({
            label: service,
            data: (data.metrics[service] ?? []).map((metric) => metric.logsCount),
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
            {isEmpty && (
                <Alert variant="outlined" severity="info">
                    No logs received in the last 60 seconds
                </Alert>
            )}

            {/* only show the graph if data and timestamps are valid */}
            {!isLoading && data && data.timestamps.length > 0 && (
                <LineChart
                    xAxis={[
                        {
                            data: data.timestamps,
                            scaleType: "time",
                            valueFormatter: (v) => new Date(v).toLocaleTimeString(),
                            label: "Time",
                            tickInterval: data.timestamps.filter((_, i) => i % 5 === 0), // longer lines at x axis only appear for every 5 seconds
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
