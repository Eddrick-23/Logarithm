import IngestionGraph from "../components/IngestionGraph";
import { Grid, Box, Stack, Typography, Button } from "@mui/material";
import ServiceOverview from "../components/ServiceOverview";
import ServiceError from "../components/ServiceError";
import { useDashboard } from "../hooks/useMetrics";
import RefreshIcon from "@mui/icons-material/Refresh";
import NatsQueueDepth from "../components/NatsQueueDepth";
import PulsingCircle from "../components/PulsingCircle";

const STATUS_CONFIG = {
    live: { label: "LIVE", colour: "success.main" },
    connecting: { label: "CONNECTING", colour: "warning.main" },
    error: { label: "OFFLINE", colour: "error.main" },
};

function getStatus(isLoading: boolean, isError: boolean) {
    if (isLoading) return STATUS_CONFIG.connecting;
    if (isError) return STATUS_CONFIG.error;
    return STATUS_CONFIG.live;
}

export default function Dashboard() {
    const { isLoading, isError, refetch } = useDashboard();
    const status = getStatus(isLoading, isError);

    return (
        <Box sx={{ bgcolor: "background.default", p: 2 }}>
            {/* Header */}
            <Stack direction="row" sx={{ alignItems: "center", justifyContent: "space-between", mb: 2 }}>
                <Typography sx={{ fontSize: 13, fontWeight: 600, letterSpacing: "0.04em", color: "text.secondary" }}>
                    SYSTEM OVERVIEW
                </Typography>
                <Stack direction="row" sx={{ alignItems: "center" }} spacing={1.5}>
                    {isError && !isLoading && (
                        <Button
                            size="small"
                            onClick={() => refetch()}
                            startIcon={<RefreshIcon sx={{ fontSize: 14 }} />}
                            sx={{
                                fontSize: 10,
                                fontWeight: 700,
                                letterSpacing: "0.06em",
                                color: "error.main",
                                py: 0.25,
                                px: 1,
                                minWidth: "auto",
                                lineHeight: 1.4,
                                "&:hover": { bgcolor: "error.main", color: "error.contrastText" },
                            }}
                        >
                            RETRY
                        </Button>
                    )}
                    <Stack direction="row" sx={{ alignItems: "center" }} spacing={1}>
                        <PulsingCircle color={status.colour} />
                        <Typography
                            sx={{
                                fontSize: 10,
                                fontWeight: 700,
                                letterSpacing: "0.1em",
                                color: status.colour,
                            }}
                        >
                            {status.label}
                        </Typography>
                    </Stack>
                </Stack>
            </Stack>

            {/* Stat cards */}
            <Box sx={{ mb: 2 }}>
                <ServiceOverview isLoading={isLoading} />
            </Box>

            {/* Chart + Latency */}
            <Grid container spacing={2} sx={{ mb: 2, alignItems: "stretch" }}>
                <Grid size="grow">
                    <IngestionGraph isLoading={isLoading} isError={isError} />
                </Grid>
                {/* for 1200px <= size < 1536px, size assigned is larger to fit the ServiceError without overflowing
                    for size >= 1536px, size assigned is smaller since there is sufficient space to fit ServiceError without overflowing */}
                <Grid size={{ lg: 3.25, xl: 2.75 }}>
                    <ServiceError />
                </Grid>
            </Grid>

            {/* nats queue depth graphs */}
            <NatsQueueDepth />
        </Box>
    );
}
