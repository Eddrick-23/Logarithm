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

    const isEmpty = !isLoading && !isError && services.length === 0;

    return (
        <Box sx={{ ...card }}>
            <Typography sx={sectionLabel}>Ingestion Throughput - Last 60s</Typography>

            <LastUpdated timestamp={lastUpdatedAt} refreshIntervalMs={INGESTION_GRAPH_REFETCH_INTERVAL_MS} />

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

            {/* no logs received from api */}
            {isEmpty && (
                // add a gap between LastUpdatedAt and alert bar
                <Box sx={{ mt: 1.5 }}>
                    <Alert variant="outlined" severity="info">
                        No logs received in the last 60 seconds
                    </Alert>
                </Box>
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
