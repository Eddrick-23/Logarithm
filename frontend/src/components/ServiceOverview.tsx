import { Box, Typography, Grid, Skeleton } from "@mui/material";
import { card, sectionLabel, statValue } from "../theme/tokens";
import { formatNumber } from "../utils/utils";
import { useErrorRateMetrics, useLogRateStats, useStorageInfoMetrics } from "../hooks/useMetrics";
import {
    ERROR_RATE_METRICS_REFETCH_INTERVAL_MS,
    LOG_RATE_METRICS_REFETCH_INTERVAL_MS,
    STORAGE_INFO_METRICS_REFETCH_INTERVAL_MS,
} from "../api/metricsApi";
import { LastUpdated } from "./LastUpdated";
import { memo } from "react";

interface ServiceOverviewProps {
    isLoading: boolean;
}

const StatCard = memo(function StatCard({
    label,
    value,
    unit,
    delta,
    deltaColor = "text.secondary",
    lastUpdated,
    refetchIntervalMs,
}: {
    label: string;
    value: string | number;
    unit?: string;
    delta: string;
    deltaColor?: string;
    lastUpdated: number;
    refetchIntervalMs: number;
}) {
    return (
        <Box sx={card}>
            <Typography sx={sectionLabel}>{label}</Typography>
            <Typography sx={{ ...statValue }}>
                {value}
                {unit && (
                    <Box component="span" sx={{ fontSize: 18, color: "text.secondary" }}>
                        {unit}
                    </Box>
                )}
            </Typography>
            <Typography sx={{ fontSize: 13, fontWeight: 500, color: deltaColor }}>{delta}</Typography>

            {/* last updated display with refetch interval */}
            <LastUpdated timestamp={lastUpdated} refreshIntervalMs={refetchIntervalMs} />
        </Box>
    );
});

export default memo(function ServiceOverview({ isLoading }: ServiceOverviewProps) {
    const { data: logRateStats, dataUpdatedAt: logRateUpdatedAt } = useLogRateStats();
    const { data: errorRateMetrics, dataUpdatedAt: errorRateMetricsUpdatedAt } = useErrorRateMetrics();
    const { data: storageInfoMetrics, dataUpdatedAt: storageInfoMetricsUpdatedAt } = useStorageInfoMetrics();

    let logRate: string | number = 0;
    if (logRateStats?.currentRate !== undefined) {
        logRate = formatNumber(logRateStats.currentRate);
    }

    // log rate conditional display
    let logDeltaText = "—";
    let logDeltaColour = "text.secondary";
    if (logRateStats?.avgRate === 0) {
        logDeltaText = "No logs received in the past minute.";
        logDeltaColour = "error.main";
    } else if (logRateStats?.ratio !== undefined) {
        const percentChange = (logRateStats.ratio - 1) * 100;
        const absChange = Math.abs(percentChange).toFixed(0);

        if (percentChange > 0) {
            logDeltaText = `↑ ${absChange}% vs avg`;
            logDeltaColour = "success.main";
        } else if (percentChange < 0) {
            logDeltaText = `↓ ${absChange}% vs avg`;
            logDeltaColour = "error.main";
        } else {
            logDeltaText = `~ 0% vs avg`;
            logDeltaColour = "text.secondary";
        }
    }

    // error rate conditional display (bucket into severity)
    // < 1%: normal
    // < 5%: elevated
    // otherwise: critical
    let errorRateValue: string | number = "-";
    let errorDeltaText = "—";
    let errorDeltaColour = "text.secondary";
    if (errorRateMetrics?.currentRate !== undefined) {
        const ratePercent = errorRateMetrics.currentRate;
        errorRateValue = ratePercent.toFixed(2);

        if (ratePercent < 1) {
            errorDeltaText = "● Normal";
            errorDeltaColour = "success.main";
        } else if (ratePercent < 5) {
            errorDeltaText = "● Elevated";
            errorDeltaColour = "warning.main";
        } else {
            errorDeltaText = "● Critical";
            errorDeltaColour = "error.main";
        }
    }

    if (isLoading) {
        return (
            <Grid container spacing={2}>
                {[1, 2, 3].map((skeletonKey) => (
                    <Grid size={4} key={skeletonKey}>
                        <Skeleton variant="rounded" height={110} sx={{ borderRadius: 2 }} />
                    </Grid>
                ))}
            </Grid>
        );
    }

    return (
        <Grid container spacing={2}>
            <Grid size={4}>
                <StatCard
                    label="Logs / sec"
                    value={logRate}
                    delta={logDeltaText}
                    deltaColor={logDeltaColour}
                    lastUpdated={logRateUpdatedAt}
                    refetchIntervalMs={LOG_RATE_METRICS_REFETCH_INTERVAL_MS}
                />
            </Grid>
            <Grid size={4}>
                <StatCard
                    label="Error Rate"
                    value={errorRateValue}
                    unit="%"
                    delta={errorDeltaText}
                    deltaColor={errorDeltaColour}
                    lastUpdated={errorRateMetricsUpdatedAt}
                    refetchIntervalMs={ERROR_RATE_METRICS_REFETCH_INTERVAL_MS}
                />
            </Grid>
            <Grid size={4}>
                <StatCard
                    label="Storage"
                    value={storageInfoMetrics?.value ?? "-"}
                    unit={storageInfoMetrics?.unit}
                    delta={storageInfoMetrics?.delta ?? "—"}
                    deltaColor={storageInfoMetrics?.deltaColour ?? "text.secondary"}
                    lastUpdated={storageInfoMetricsUpdatedAt}
                    refetchIntervalMs={STORAGE_INFO_METRICS_REFETCH_INTERVAL_MS}
                />
            </Grid>
        </Grid>
    );
});
