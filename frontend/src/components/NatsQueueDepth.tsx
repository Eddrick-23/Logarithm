import { useMemo } from "react";
import { useNatsQueueDepthGraphMetrics } from "../hooks/useMetrics";
import { Stack } from "@mui/material";
import NatsQueueDepthGraph from "./NatsQueueDepthGraph";

export default function NatsQueueDepth() {
    const { data, isLoading, isError, dataUpdatedAt } = useNatsQueueDepthGraphMetrics();

    const xAxisData = useMemo(() => data?.timestamps.map((ts) => new Date(ts)) ?? [], [data]);
    const numPendingData = useMemo(() => [{ data: data?.numPending ?? [] }], [data]);
    const numAckPendingdData = useMemo(() => [{ data: data?.numAckPending ?? [] }], [data]);

    return (
        <Stack direction="row" spacing={2} sx={{ width: "100%" }}>
            <NatsQueueDepthGraph
                title="Num Pending - number of logs waiting in the queue to be delivered"
                xAxisData={xAxisData}
                seriesData={numPendingData}
                isLoading={isLoading}
                isError={isError}
                dataUpdatedAt={dataUpdatedAt}
            />
            <NatsQueueDepthGraph
                title="Num Ack Pending - number of logs currently being processed"
                xAxisData={xAxisData}
                seriesData={numAckPendingdData}
                isLoading={isLoading}
                isError={isError}
                dataUpdatedAt={dataUpdatedAt}
            />
        </Stack>
    );
}
