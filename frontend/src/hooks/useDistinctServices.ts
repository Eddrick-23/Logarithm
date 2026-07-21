import { useQuery } from "@tanstack/react-query";
import { fetchDistinctServices } from "../api/servicesApi";

export const useDistinctServices = () => {
    return useQuery({
        queryKey: ["distinctServices"],
        queryFn: fetchDistinctServices,
        refetchInterval: 1000 * 2, // refreshes every 2 s
    });
};
