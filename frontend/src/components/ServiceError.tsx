import { Box, LinearProgress, Skeleton, Tooltip, Typography } from "@mui/material";
import { card, sectionLabel } from "../theme/tokens";
import { useTopServiceErrorsStats } from "../hooks/useMetrics";
import type { TopServiceErrorsStats } from "../types/Metric";
import { formatNumber } from "../utils/utils";
import { TOP_SERVICE_ERRORS_STATS_REFETCH_INTERVAL_MS } from "../api/metricsApi";
import { LastUpdated } from "./LastUpdated";

const NUM_SERVICES = 5;
const SEVERITY_THRESHOLDS = [
    { min: 80, colour: "#ef5350" }, // critical  — red
    { min: 60, colour: "#ffa726" }, // high      — orange
    { min: 40, colour: "#fbc02d" }, // medium    — yellow
    { min: 20, colour: "#42a5f5" }, // low       — blue
    { min: 0, colour: "#607d8b" }, // minimal   — grey
] as const;

function getSeverityColour(errorRate: number): string {
    return SEVERITY_THRESHOLDS.find(({ min }) => errorRate >= min)!.colour;
}

interface ServiceErrorRowProps {
    serviceName: string;
    totalErrors: number;
    errorRate: number;
}

function ServiceErrorRow({ serviceName, totalErrors, errorRate }: ServiceErrorRowProps) {
    return (
        <Box sx={{ display: "flex", alignItems: "center", py: 1.5 }}>
            <Tooltip title={serviceName} placement="top-start">
                <Typography
                    noWrap
                    sx={{ width: { lg: 75, xl: 120 }, fontSize: "0.85rem", flexShrink: 0, cursor: "default" }}
                >
                    {serviceName}
                </Typography>
            </Tooltip>
            <Box sx={{ flexGrow: 1, minWidth: { lg: 30, xl: 60 }, mx: 2 }}>
                <LinearProgress
                    variant="determinate"
                    value={errorRate}
                    sx={{
                        height: 6,
                        borderRadius: 3,
                        backgroundColor: "rgba(255, 255, 255, 0.08)",
                        "& .MuiLinearProgress-bar": {
                            backgroundColor: getSeverityColour(errorRate),
                            borderRadius: 3,
                        },
                    }}
                />
            </Box>
            <Box sx={{ display: "flex", flexDirection: "column", alignItems: "flex-end", flexShrink: 0, width: 60 }}>
                <Tooltip title={totalErrors.toLocaleString()} placement="top">
                    <Typography sx={{ fontWeight: "bold", fontSize: "0.85rem" }}>
                        {formatNumber(totalErrors)}
                    </Typography>
                </Tooltip>
                <Typography sx={{ fontSize: "0.75rem", color: "#8b949e" }}>{errorRate.toFixed(2)}%</Typography>
            </Box>
        </Box>
    );
}

export default function ServiceError() {
    const { data, isLoading, isError, dataUpdatedAt } = useTopServiceErrorsStats();

    return (
        <Box sx={{ ...card, height: "100%" }}>
            {/* header */}
            <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: 2 }}>
                <Typography sx={sectionLabel}>Top errors by service</Typography>
                <Typography variant="caption" sx={{ color: "#8b949e", fontSize: "0.85rem", mb: 1 }}>
                    last 1h
                </Typography>
            </Box>

            {/* last updated display with refetch interval */}
            <LastUpdated timestamp={dataUpdatedAt} refreshIntervalMs={TOP_SERVICE_ERRORS_STATS_REFETCH_INTERVAL_MS} />

            {/* create loading skeleton bars to simulate loading service errors */}
            {isLoading && (
                <Box sx={{ display: "flex", flexDirection: "column", gap: 1.5, mt: 1 }}>
                    {Array.from({ length: NUM_SERVICES }).map((_, i) => (
                        <Skeleton key={i} variant="rectangular" height={36} sx={{ borderRadius: 1 }} />
                    ))}
                </Box>
            )}

            {/* no errors found in past hour */}
            {!isLoading && !isError && (!data || data.length === 0) && (
                <Box sx={{ py: 3, textAlign: "center" }}>
                    <Typography variant="body2" sx={{ color: "#8b949e" }}>
                        No errors recorded in the last hour.
                    </Typography>
                </Box>
            )}

            {/* display rows */}
            {!isLoading && !isError && data && data.length > 0 && (
                <Box>
                    {data.map((item: TopServiceErrorsStats) => (
                        <ServiceErrorRow
                            key={item.serviceName}
                            serviceName={item.serviceName}
                            totalErrors={item.totalErrors}
                            errorRate={item.errorRate}
                        />
                    ))}
                </Box>
            )}
        </Box>
    );
}
