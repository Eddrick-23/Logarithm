import { Box, Typography, Grid, Skeleton } from "@mui/material";
import { card, sectionLabel, statValue } from "../theme/tokens";
import { useIngestionMetrics } from "../hooks/useMetrics";
import { formatNumber } from "../utils/utils";

interface ServiceOverviewProps {
    errorRate: number;
    numOfLiveServices: number;
}

function StatCard({
    label,
    value,
    unit,
    delta,
    deltaColor = "text.secondary",
}: {
    label: string;
    value: string | number;
    unit?: string;
    delta: string;
    deltaColor?: string;
}) {
    return (
        <Box sx={card}>
            <Typography sx={sectionLabel}>{label}</Typography>
            <Typography sx={{ ...statValue }}>
                {value}
                {unit && (
                    <Box component="span" sx={{ fontSize: 18, color: "text.disabled" }}>
                        {unit}
                    </Box>
                )}
            </Typography>
            <Typography sx={{ fontSize: 13, fontWeight: 500, color: deltaColor }}>{delta}</Typography>
        </Box>
    );
}

export default function ServiceOverview({ errorRate, numOfLiveServices }: ServiceOverviewProps) {
    const { data, isLoading } = useIngestionMetrics();

    const stats = data?.logStats;
    let logRate: string | number = 0;
    if (stats?.currentRate !== undefined) {
        logRate = formatNumber(stats.currentRate);
    }

    let logDeltaText = "";
    let logDeltaColour = "text.secondary";
    if (stats?.ratio !== undefined) {
        const percentChange = (stats.ratio - 1) * 100;
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
                <StatCard label="Logs / sec" value={logRate} delta={logDeltaText} deltaColor={logDeltaColour} />
            </Grid>
            <Grid size={4}>
                <StatCard label="Error Rate" value={errorRate} unit="%" delta="● Normal" deltaColor="success.main" />
            </Grid>
            <Grid size={4}>
                <StatCard
                    label="Services"
                    value={numOfLiveServices}
                    unit=" live"
                    delta="All healthy"
                    deltaColor="success.main"
                />
            </Grid>
        </Grid>
    );
}
