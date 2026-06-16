import IngestionGraph from "../components/IngestionGraph";
import { Grid, Box, Stack, Typography } from "@mui/material";
import ServiceOverview from "../components/ServiceOverview";
import ServiceError from "../components/ServiceError";
import LiveTailLogs from "../components/LiveTailLogs";
import { pulseSx } from "../theme/tokens";
import { useIngestionMetrics } from "../hooks/useMetrics";

export default function Dashboard() {
    const { data, isLoading, isError, refetch } = useIngestionMetrics();

    return (
        <Box sx={{ bgcolor: "background.default", minHeight: "100vh", p: 2 }}>
            {/* Header */}
            <Stack direction="row" sx={{ alignItems: "center", justifyContent: "space-between", mb: 2 }}>
                <Typography sx={{ fontSize: 13, fontWeight: 600, letterSpacing: "0.04em", color: "text.secondary" }}>
                    SYSTEM OVERVIEW
                </Typography>
                <Stack direction="row" sx={{ alignItems: "center" }} spacing={1}>
                    <Box sx={{ ...pulseSx, color: "success.main" }} />
                    <Typography
                        sx={{
                            fontSize: 10,
                            fontWeight: 700,
                            letterSpacing: "0.1em",
                            color: "text.disabled",
                        }}
                    >
                        LIVE
                    </Typography>
                </Stack>
            </Stack>

            {/* Stat cards */}
            <Box sx={{ mb: 2 }}>
                <ServiceOverview data={data?.logStats} isLoading={isLoading} errorRate={0.3} numOfLiveServices={7} />
            </Box>

            {/* Chart + Latency */}
            <Grid container spacing={2} sx={{ mb: 2, alignItems: "stretch" }}>
                <Grid size="grow">
                    <IngestionGraph data={data?.graph} isLoading={isLoading} isError={isError} refetch={refetch} />
                </Grid>
                {/* for 1200px <= size < 1536px, size assigned is larger to fit the ServiceError without overflowing
                    for size >= 1536px, size assigned is smaller since there is sufficient space to fit ServiceError without overflowing */}
                <Grid size={{ lg: 3.25, xl: 2.75 }}>
                    <ServiceError />
                </Grid>
            </Grid>

            {/* Live logs */}
            <LiveTailLogs />
        </Box>
    );
}
