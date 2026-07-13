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

// set min difference between each number on y-axis to be 1 and round off all numbers on y-axis to whole numbers
const yAxis = [{ min: 0, tickMinStep: 1, valueFormatter: (value: number) => Math.round(value).toString() }];

export default function NatsQueueDepthGraph({
    title,
    xAxisData,
    seriesData,
    isLoading,
    isError,
    dataUpdatedAt,
}: NatsQueueDepthGraphProps) {
    const hasData = xAxisData && xAxisData.length > 0;
    const isEmpty = !isLoading && !isError && !hasData;
    const isHardError = !isLoading && isError && !hasData;
    const isStale = !isLoading && isError && hasData;

    return (
        <Box sx={{ ...card, flex: 1 }}>
            <Typography sx={sectionLabel}>{title}</Typography>
            <LastUpdated timestamp={dataUpdatedAt} refreshIntervalMs={REFETCH_INTERVAL_MS} />

            {/* rectangular skeleton box to signify loading of graph */}
            {isLoading && (
                <Skeleton variant="rectangular" width="100%" height={CHART_HEIGHT} sx={{ borderRadius: 2 }} />
            )}

            {/* stale data: show last known data while attempting to refetch */}
            {isStale && (
                // add a gap between LastUpdatedAt and alert bar
                <Box sx={{ mt: 1.5 }}>
                    <Alert variant="outlined" severity="warning">
                        Showing last known data.
                    </Alert>
                </Box>
            )}

            {isHardError && (
                <Box sx={{ my: 2 }}>
                    <Alert severity="error">Error fetching data.</Alert>
                </Box>
            )}

            {isEmpty && (
                <Box sx={{ my: 2 }}>
                    <Alert severity="info">No data yet.</Alert>
                </Box>
            )}

            {/* after a while, there will be data recorded to be displayed */}
            {!isLoading && hasData && (
                <LineChart
                    skipAnimation
                    xAxis={[
                        {
                            data: xAxisData,
                            scaleType: "time",
                            valueFormatter: (date) => date.toLocaleTimeString(),
                        },
                    ]}
                    yAxis={yAxis}
                    series={seriesData}
                    height={CHART_HEIGHT}
                />
            )}
        </Box>
    );
}
