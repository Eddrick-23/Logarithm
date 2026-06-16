import { LineChart } from "@mui/x-charts/LineChart";
import { useMemo, useState } from "react";
import { Alert, Box, Checkbox, Chip, FormControlLabel, FormGroup, Skeleton, Stack, Typography } from "@mui/material";
import { card, sectionLabel } from "../theme/tokens";
import ErrorBanner from "./ErrorBanner";
import type { IngestionGraphData } from "../types/Metric";

interface IngestionGraphProps {
    data?: IngestionGraphData;
    isLoading: boolean;
    isError: boolean;
    refetch: () => void;
}

const CHART_HEIGHT = 400;

// automatically generates the colour based on golden angle formula
const generateColour = (index: number) => `hsl(${(index * 137.5) % 360}, 70%, 50%)`;

export default function IngestionGraph({ data, isLoading, isError, refetch }: IngestionGraphProps) {
    const [hiddenServices, setHiddenServices] = useState<Set<string>>(new Set());

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

    const toggleService = (service: string) => {
        setHiddenServices((prev) => {
            const currServices = new Set(prev);
            if (currServices.has(service)) {
                currServices.delete(service);
            } else {
                currServices.add(service);
            }
            return currServices;
        });
    };

    const allSelected = services.every((service) => !hiddenServices.has(service));
    const toggleAll = () => {
        setHiddenServices(allSelected ? new Set(services) : new Set());
    };

    const isEmpty = !isLoading && !isError && services.length === 0;

    return (
        <Box sx={{ ...card }}>
            <Typography sx={sectionLabel}>Ingestion Throughput - Last 60s</Typography>

            {/* filters to choose which services to track on ingestion graph */}
            {!isLoading && services.length > 0 && (
                <Box sx={{ mb: 2 }}>
                    <Stack direction="row" sx={{ flexWrap: "wrap", gap: 2.5, alignItems: "center" }}>
                        <Chip
                            label="All"
                            size="small"
                            variant={allSelected ? "filled" : "outlined"}
                            onClick={toggleAll}
                            sx={{ fontWeight: 600 }}
                        />
                        <FormGroup row>
                            {services.map((service) => (
                                <FormControlLabel
                                    key={service}
                                    control={
                                        <Checkbox
                                            size="small"
                                            checked={!hiddenServices.has(service)}
                                            onChange={() => toggleService(service)}
                                            sx={{
                                                color: serviceColours[service],
                                                "&.Mui-checked": { color: serviceColours[service] },
                                            }}
                                        />
                                    }
                                    label={
                                        <Typography variant="body2" noWrap>
                                            {service}
                                        </Typography>
                                    }
                                />
                            ))}
                        </FormGroup>
                    </Stack>
                </Box>
            )}

            {/* rectangular skeleton box to signify loading of graph */}
            {isLoading && (
                <Skeleton variant="rectangular" width="100%" height={CHART_HEIGHT} sx={{ borderRadius: 2 }} />
            )}

            {/* error banner */}
            {!isLoading && isError && <ErrorBanner service="server" handleReconnect={refetch} />}

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
