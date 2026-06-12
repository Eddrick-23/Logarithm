import { useQuery } from "@tanstack/react-query";
import { fetchIngestionMetrics } from "../api/metricsApi";

export const useIngestionMetrics = () => {
    return useQuery({
        queryKey: ["ingestionMetrics"],
        queryFn: fetchIngestionMetrics,
        staleTime: 1000 * 60, // 1 minute
        refetchInterval: 1000 * 5, // refetch every 5 seconds
    });
};
