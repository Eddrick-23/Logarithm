import { Box, Typography, Grid } from "@mui/material";
import { card, sectionLabel, statValue } from "../theme/tokens";

interface ServiceOverviewProps {
    logsPerSecond: number;
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

export default function ServiceOverview({ logsPerSecond, errorRate, numOfLiveServices }: ServiceOverviewProps) {
    return (
        <Grid container spacing={2}>
            <Grid size={4}>
                <StatCard
                    label="Logs / sec"
                    value={logsPerSecond / 1000}
                    unit="k"
                    delta="↑ 12% vs avg"
                    deltaColor="success.main"
                />
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
