import { LineChart } from "@mui/x-charts/LineChart";
import { useMemo, useState } from "react";
import { Alert, Box, Skeleton, Typography } from "@mui/material";
import { card, sectionLabel } from "../theme/tokens";
import { INGESTION_GRAPH_REFETCH_INTERVAL_MS } from "../api/metricsApi";
import { LastUpdated } from "./LastUpdated";
import { ServiceFilters } from "./ServiceFilters";
import { useIngestionGraphMetrics } from "../hooks/useMetrics";

interface IngestionGraphProps {
    isLoading: boolean;
    isError: boolean;
}

const CHART_HEIGHT = 400;

// automatically generates the colour based on golden angle formula
const generateColour = (index: number) => `hsl(${(index * 137.5) % 360}, 70%, 50%)`;
const formatTime = (v: number) => new Date(v).toLocaleTimeString();

export default function IngestionGraph({ isLoading, isError }: IngestionGraphProps) {
    const [hiddenServices, setHiddenServices] = useState<Set<string>>(new Set());
    const { data, dataUpdatedAt: lastUpdatedAt } = useIngestionGraphMetrics();

    // use to store the service names in ascending order
    const servicesKey = useMemo(
        () =>
            Object.keys(data?.metrics ?? {})
                .sort()
                .join(","),
        [data],
    );

    const services = useMemo(
        () => Object.keys(data?.metrics ?? {}),
        [servicesKey], // only recompute when the actual set of service names changes
    );

    const serviceColours = useMemo(
        // map service name to colour based on the idx
        () => Object.fromEntries(services.map((s, i) => [s, generateColour(i)])),
        [services],
    );

    const series = useMemo(() => {
        if (!data) return [];
        return services
            .filter((service) => !hiddenServices.has(service)) // only show services which are not hidden
            .map((service) => ({
                label: service,
                data: (data.metrics[service] ?? []).map((metric) => metric.logsCount),
                color: serviceColours[service],
            }));
    }, [data, services, hiddenServices, serviceColours]);

    const tickInterval = useMemo(() => data?.timestamps.filter((_, i) => i % 5 === 0) ?? [], [data?.timestamps]);
    const xAxis = useMemo(
        () => [
            {
                data: data?.timestamps ?? [],
                scaleType: "time" as const,
                valueFormatter: formatTime,
                label: "Time",
                tickInterval,
            },
        ],
        [data?.timestamps, tickInterval],
    );
    const yAxis = useMemo(() => [{ min: 0, label: "Logs / sec" }], []);

    const hasData = series.length > 0;
    const isEmpty = !isLoading && !isError && !hasData;
    const isHardError = !isLoading && isError && !hasData;
    const isStale = !isLoading && isError && hasData;

    return (
        <Box sx={{ ...card, height: "100%" }}>
            <Typography sx={sectionLabel}>Ingestion Throughput - Last 60s</Typography>

            <LastUpdated timestamp={lastUpdatedAt} refreshIntervalMs={INGESTION_GRAPH_REFETCH_INTERVAL_MS} />

            {/* stale data: show last known data while attempting to refetch */}
            {isStale && (
                // add a gap between LastUpdatedAt and alert bar
                <Box sx={{ mt: 1.5 }}>
                    <Alert variant="outlined" severity="warning">
                        Showing last known data.
                    </Alert>
                </Box>
            )}

            {/* filters to choose which services to track on ingestion graph */}
            {!isLoading && services.length > 0 && (
                <ServiceFilters
                    services={services}
                    serviceColours={serviceColours}
                    hiddenServices={hiddenServices}
                    setHiddenServices={setHiddenServices}
                />
            )}

            {/* rectangular skeleton box to signify loading of graph */}
            {isLoading && (
                <Skeleton variant="rectangular" width="100%" height={CHART_HEIGHT} sx={{ borderRadius: 2 }} />
            )}

            {/* hard failure: websocket + rest api fetch both failed, nothing to display */}
            {isHardError && (
                // add a gap between LastUpdatedAt and alert bar
                <Box sx={{ mt: 1.5 }}>
                    <Alert variant="outlined" severity="error">
                        Unable to load ingestion graph data.
                    </Alert>
                </Box>
            )}

            {/* no logs received from api in the past minute */}
            {isEmpty && (
                // add a gap between LastUpdatedAt and alert bar
                <Box sx={{ mt: 1.5 }}>
                    <Alert variant="outlined" severity="info">
                        No logs received in the past minute.
                    </Alert>
                </Box>
            )}

            {/* only show the graph if there are timestamps found */}
            {!isLoading && data && data?.timestamps.length > 0 && (
                <LineChart
                    xAxis={xAxis}
                    yAxis={yAxis} // set min to 0 so that y starts from 0
                    series={series}
                    height={CHART_HEIGHT}
                />
            )}
        </Box>
    );
}
