import { useMemo } from "react";
import { useNatsQueueDepthGraphMetrics } from "../hooks/useMetrics";
import { Stack } from "@mui/material";
import NatsQueueDepthGraph from "./NatsQueueDepthGraph";
import type { LineChartXAxis } from "../types/LineChart";
import { formatTime } from "../utils/utils";

interface NatsQueueDepthProps {
    isLoading: boolean;
    isError: boolean;
}

export default function NatsQueueDepth({ isLoading, isError }: NatsQueueDepthProps) {
    const { data, dataUpdatedAt } = useNatsQueueDepthGraphMetrics();

    const xAxis: LineChartXAxis = useMemo(
        () => [
            {
                data: data?.timestamps ?? [],
                scaleType: "time" as const,
                valueFormatter: formatTime,
                label: "Time",
            },
        ],
        [data?.timestamps],
    );
    const numPendingData = useMemo(() => [{ data: data?.numPending ?? [] }], [data]);
    const numAckPendingdData = useMemo(() => [{ data: data?.numAckPending ?? [] }], [data]);
    const numRedeliveredData = useMemo(() => [{ data: data?.numRedelivered ?? [] }], [data]);

    return (
        <Stack spacing={2}>
            <NatsQueueDepthGraph
                title="Num Pending - number of logs waiting in the queue to be delivered"
                xAxis={xAxis}
                seriesData={numPendingData}
                isLoading={isLoading}
                isError={isError}
                dataUpdatedAt={dataUpdatedAt}
            />

            <Stack direction="row" spacing={2} sx={{ width: "100%" }}>
                <NatsQueueDepthGraph
                    title="Num Ack Pending - number of logs currently being processed"
                    xAxis={xAxis}
                    seriesData={numAckPendingdData}
                    isLoading={isLoading}
                    isError={isError}
                    dataUpdatedAt={dataUpdatedAt}
                />
                <NatsQueueDepthGraph
                    title="Num Redelivered - number of times logs has been resent"
                    xAxis={xAxis}
                    seriesData={numRedeliveredData}
                    isLoading={isLoading}
                    isError={isError}
                    dataUpdatedAt={dataUpdatedAt}
                />
            </Stack>
        </Stack>
    );
}
