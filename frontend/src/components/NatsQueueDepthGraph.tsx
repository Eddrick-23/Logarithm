import { Alert, Box, Skeleton, Typography } from "@mui/material";
import { card, sectionLabel } from "../theme/tokens";
import { LastUpdated } from "./LastUpdated";
import { LineChart } from "@mui/x-charts";

const REFETCH_INTERVAL_MS = 15000;
const CHART_HEIGHT = 400;

interface SeriesConfig {
    data: number[];
}

interface NatsQueueDepthGraphProps {
    title: string;
    xAxisData: Date[];
    seriesData: SeriesConfig[];
    isLoading: boolean;
    isError: boolean;
    dataUpdatedAt: number;
}

export default function NatsQueueDepthGraph({
    title,
    xAxisData,
    seriesData,
    isLoading,
    isError,
    dataUpdatedAt,
}: NatsQueueDepthGraphProps) {
    return (
        <Box sx={{ ...card, flex: 1 }}>
            <Typography sx={sectionLabel}>{title}</Typography>
            <LastUpdated timestamp={dataUpdatedAt} refreshIntervalMs={REFETCH_INTERVAL_MS} />

            {/* rectangular skeleton box to signify loading of graph */}
            {isLoading && (
                <Skeleton variant="rectangular" width="100%" height={CHART_HEIGHT} sx={{ borderRadius: 2 }} />
            )}

            {isError && (
                <Box sx={{ my: 2 }}>
                    <Alert severity="error">Error fetching data.</Alert>
                </Box>
            )}

            {!isLoading && xAxisData && xAxisData.length > 0 && (
                <LineChart
                    xAxis={[
                        { data: xAxisData, scaleType: "time", valueFormatter: (date) => date.toLocaleTimeString() },
                    ]}
                    yAxis={[{ min: 0 }]}
                    series={seriesData}
                    height={CHART_HEIGHT}
                />
            )}
        </Box>
    );
}
