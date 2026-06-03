import { useQuery } from "@tanstack/react-query";
import { fetchDistinctServices } from "../api/servicesApi";

export const useDistinctServices = () => {
    return useQuery({
        queryKey: ["distinctServices"],
        queryFn: fetchDistinctServices,
        staleTime: 1000 * 60 * 5, // Data stays "fresh" for 5 minutes
    });
};
