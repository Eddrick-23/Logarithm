import IngestionGraph from "../components/IngestionGraph";
import { Grid, Box, Stack, Typography } from "@mui/material";
import ServiceOverview from "../components/ServiceOverview";
import LatencyView from "../components/ServiceError";
import LiveTailLogs from "../components/LiveTailLogs";
import { pulseSx } from "../theme/tokens";

export default function Dashboard() {
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
                <ServiceOverview logsPerSecond={4200} errorRate={0.3} numOfLiveServices={7} />
            </Box>

            {/* Chart + Latency */}
            <Grid container spacing={2} sx={{ mb: 2, alignItems: "stretch" }}>
                <Grid size="grow">
                    <IngestionGraph />
                </Grid>
                <Grid size={2.75}>
                    <LatencyView />
                </Grid>
            </Grid>

            {/* Live logs */}
            <LiveTailLogs />
        </Box>
    );
}
