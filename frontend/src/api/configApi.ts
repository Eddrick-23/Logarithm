import axios from "axios";
import type { Config } from "../types/Config";

export const fetchConfig = async (): Promise<Config> => {
    const response = await axios.get<Config>("/api/config");
    return response.data;
};
