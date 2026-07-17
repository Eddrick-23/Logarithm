import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import { parseEnv } from "./src/config/env.schema";

// https://vite.dev/config/

export default defineConfig(({ mode }) => {
    const rawEnv = loadEnv(mode, process.cwd(), "");
    const env = parseEnv(rawEnv);

    const apiUrl = env.VITE_API_URL;
    console.log("My API url is: ", apiUrl);
    const websocketUrl = env.VITE_WEBSOCKET_URL;
    console.log("My Websocket url is: ", websocketUrl);

    return {
        plugins: [
            tanstackRouter({
                target: "react",
                autoCodeSplitting: true,
            }),
            react(),
        ],
        server: {
            host: true,
            port: 5173,
            watch: {
                usePolling: true,
            },

            proxy: {
                "/api": {
                    target: apiUrl,
                    changeOrigin: true,
                    secure: false,
                },
                "/ws": {
                    target: websocketUrl,
                    ws: true,
                },
            },
        },
        test: {
            globals: true,
            environment: "jsdom",
            setupFiles: ["./src/tests/setupTests.ts"],
        },
    };
});
