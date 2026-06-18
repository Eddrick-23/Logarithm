import { useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchTopServiceErrorsStats, fetchIngestionMetrics } from "../api/metricsApi";
import { useCallback, useEffect, useRef } from "react";

const STALE_THRESHOLD_MS = 15000; // 15s stale time

export const useIngestionMetrics = () => {
    const queryClient = useQueryClient();
    const eventSourceRef = useRef<EventSource | null>(null);
    const lastMessageRef = useRef<number>(Date.now());

    const query = useQuery({
        queryKey: ["ingestionMetrics"],
        queryFn: fetchIngestionMetrics,
        staleTime: Infinity, // SSE keeps it fresh, no need for TanStack Query to refetch
        refetchInterval: false,
    });

    const { data: connectionError } = useQuery({
        queryKey: ["ingestionMetricsConnectionError"],
        queryFn: () => false,
        initialData: false,
        staleTime: Infinity,
    });

    const openStream = useCallback(() => {
        // close existing connection before opening new conneciton
        eventSourceRef.current?.close();
        lastMessageRef.current = Date.now(); // reset watchdog clock on (re)connect

        const eventSource = new EventSource("/api/ingestion-metrics/stream");

        eventSource.addEventListener("ingestion", (e) => {
            const data = JSON.parse(e.data);
            queryClient.setQueryData(["ingestionMetrics"], data);
            queryClient.setQueryData(["ingestionMetricsConnectionError"], false);
            lastMessageRef.current = Date.now();
        });

        eventSource.addEventListener("top-service-errors", (e) => {
            const data = JSON.parse(e.data);
            queryClient.setQueryData(["topServiceErrorsStats"], data);
            queryClient.setQueryData(["ingestionMetricsConnectionError"], false);
            lastMessageRef.current = Date.now();
        });

        eventSource.onerror = () => {
            if (eventSourceRef.current?.readyState !== EventSource.OPEN) {
                queryClient.setQueryData(["ingestionMetricsConnectionError"], true);
            }
        };

        eventSource.onopen = () => {
            lastMessageRef.current = Date.now(); // update watchdog clock
            queryClient.setQueryData(["ingestionMetricsConnectionError"], false);
        };

        eventSourceRef.current = eventSource;
    }, [queryClient]);

    const connect = useCallback(async () => {
        // do REST fetch to pre load the data then (re)open SSE for live connection
        try {
            const data = await fetchIngestionMetrics();
            queryClient.setQueryData(["ingestionMetrics"], data);
            queryClient.setQueryData(["ingestionMetricsConnectionError"], false);
        } catch {
            queryClient.setQueryData(["ingestionMetricsConnectionError"], true);
        }
        openStream();
    }, [queryClient, openStream]);

    useEffect(() => {
        // open connection on mount, close on dismount
        connect();

        // watchdog: if no message received within threshold, assume connection is dead
        const interval = setInterval(() => {
            if (Date.now() - lastMessageRef.current > STALE_THRESHOLD_MS) {
                queryClient.setQueryData(["ingestionMetricsConnectionError"], true);
            }
        }, 5000);

        return () => {
            eventSourceRef.current?.close();
            clearInterval(interval);
        };
    }, [connect]);

    return {
        ...query,
        isError: query.isError || connectionError,
        refetch: connect,
    };
};

export const useTopServiceErrorsStats = () => {
    return useQuery({
        queryKey: ["topServiceErrorsStats"],
        queryFn: fetchTopServiceErrorsStats,
        staleTime: Infinity,
        refetchInterval: false,
    });
};
