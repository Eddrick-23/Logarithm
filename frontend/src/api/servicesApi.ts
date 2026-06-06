import axios from "axios";

interface ServicesResponse {
    services: string[];
}

export const fetchDistinctServices = async (): Promise<ServicesResponse> => {
    const response = await axios.get<ServicesResponse>("/api/services");
    return response.data;
};
