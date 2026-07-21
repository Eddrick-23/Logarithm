import { describe, it, expect } from "vitest";
import { parseEnv } from "../../config/env.schema";

const validHttpUrl = "http://dashboard-api:8091";
const validHttpsUrl = "https://dashboard-api:8091";
const validWebsocketUrl = "ws://dashboard-api:8091";
const validWebsocketSecureUrl = "wss://api.example.com";

describe("parseEnv", () => {
    it("accepts valid http and ws URLs", () => {
        const env = parseEnv({
            VITE_API_URL: validHttpUrl,
            VITE_WEBSOCKET_URL: validWebsocketUrl,
        });
        expect(env.VITE_API_URL).toBe(validHttpUrl);
        expect(env.VITE_WEBSOCKET_URL).toBe(validWebsocketUrl);
    });

    it("accepts https and wss", () => {
        const env = parseEnv({
            VITE_API_URL: validHttpsUrl,
            VITE_WEBSOCKET_URL: validWebsocketSecureUrl,
        });
        expect(env.VITE_API_URL).toBe(validHttpsUrl);
        expect(env.VITE_WEBSOCKET_URL).toBe(validWebsocketSecureUrl);
    });

    it("rejects a missing VITE_API_URL", () => {
        expect(() =>
            parseEnv({
                VITE_WEBSOCKET_URL: validWebsocketUrl,
            }),
        ).toThrow(/Environment validation failed/);
    });

    it("rejects a missing VITE_WEBSOCKET_URL", () => {
        expect(() =>
            parseEnv({
                VITE_API_URL: validHttpUrl,
            }),
        ).toThrow(/Environment validation failed/);
    });

    it("rejects an API URL using the ws:// protocol", () => {
        expect(() =>
            parseEnv({
                VITE_API_URL: validWebsocketUrl,
                VITE_WEBSOCKET_URL: validWebsocketUrl,
            }),
        ).toThrow(/must use protocol http/);
    });

    it("rejects a websocket URL using the http:// protocol", () => {
        expect(() =>
            parseEnv({
                VITE_API_URL: validHttpUrl,
                VITE_WEBSOCKET_URL: validHttpUrl,
            }),
        ).toThrow(/must use protocol ws/);
    });

    it("rejects a malformed API URL string", () => {
        expect(() =>
            parseEnv({
                VITE_API_URL: "not-a-url",
                VITE_WEBSOCKET_URL: validWebsocketUrl,
            }),
        ).toThrow(/must be a valid URL/);
    });

    it("rejects a malformed websocket URL string", () => {
        expect(() =>
            parseEnv({
                VITE_API_URL: validHttpUrl,
                VITE_WEBSOCKET_URL: "not-a-url",
            }),
        ).toThrow(/must be a valid URL/);
    });
});
