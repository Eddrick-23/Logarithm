import { render, screen, act, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi, describe, it, expect, beforeAll, beforeEach, afterEach, afterAll } from "vitest";
import { setupServer } from "msw/node";
import { ws } from "msw";
import LiveTailLogs from "../components/LiveTailLogs";

const connectingMessage = "Connecting to live tail server...";
const pauseMessage = "Tail paused — new logs buffering";
const connectionMessage = "";
const errorMessage = "Connection lost. Failed to connect to the live tail server.";

vi.mock("../hooks/useDistinctServices", () => ({
    useDistinctServices: () => ({
        data: { services: ["auth-service", "payment-service"] },
        isLoading: false,
    }),
}));

// mock websocket handler
const tailWs = ws.link("ws://localhost:8091/ws/logs/tail");

let sendToClient: ((data: string) => void) | null = null;
let serverCloseConnection: ((code?: number) => void) | null = null;

const server = setupServer(
    tailWs.addEventListener("connection", ({ client }) => {
        sendToClient = (data) => client.send(data);
        serverCloseConnection = (code = 1006) => client.close(code);
    }),
);

type LogBatchOverrides = {
    serviceName?: string;
    body?: string;
    severityText?: string;
    timestamp?: string;
};

const makeLogBatch = (overrides?: LogBatchOverrides) => [
    {
        serviceName: overrides?.serviceName ?? "auth-service",
        records: [
            {
                body: overrides?.body ?? "User logged in",
                severityText: overrides?.severityText ?? "INFO",
                timestamp: overrides?.timestamp ?? new Date().toISOString(),
            },
        ],
    },
];

const emitLogs = async (overrides?: LogBatchOverrides) => {
    await act(async () => sendToClient?.(JSON.stringify(makeLogBatch(overrides))));
};

/** Renders the component and waits for the WebSocket connection to be established */
const renderAndConnect = async () => {
    await act(async () => render(<LiveTailLogs />));
    await waitFor(() => expect(sendToClient).not.toBeNull());
};

beforeAll(() => server.listen());

beforeEach(() => {
    sendToClient = null;
    serverCloseConnection = null;
    vi.useFakeTimers({ shouldAdvanceTime: true });
});

afterEach(() => {
    server.resetHandlers();
    vi.useRealTimers();
    vi.restoreAllMocks();
});

afterAll(() => server.close());

describe("LiveTailLogs — connection status", () => {
    it("shows connecting bar on initial render before socket opens", async () => {
        render(<LiveTailLogs />);
        expect(screen.getByText(connectingMessage)).toBeInTheDocument();
    });

    it("hides connecting bar once socket opens", async () => {
        await renderAndConnect();
        expect(screen.queryByText(connectingMessage)).not.toBeInTheDocument();
    });

    it("shows connecting bar when server drops the connection and is retrying", async () => {
        await renderAndConnect();
        await act(async () => serverCloseConnection?.(1006));
        await waitFor(() => expect(screen.getByText(connectingMessage)).toBeInTheDocument());
    });

    it("shows error bar after max reconnect attempts are exhausted", async () => {
        await renderAndConnect();

        server.use(
            tailWs.addEventListener("connection", ({ client }) => {
                client.close(1006);
            }),
        );

        await act(async () => serverCloseConnection?.(1006));
        for (let i = 0; i < 5; i++) {
            await act(async () => vi.advanceTimersByTime(1000 * 2 ** i + 100));
        }

        await waitFor(() => expect(screen.getByText(errorMessage)).toBeInTheDocument());
    });

    it("does not reconnect on normal closure (code 1000)", async () => {
        await renderAndConnect();

        const instanceBefore = sendToClient;
        await act(async () => serverCloseConnection?.(1000));
        await act(async () => vi.advanceTimersByTime(500));

        expect(sendToClient).toBe(instanceBefore);
    });

    it("recovers to connected state after a successful reconnect", async () => {
        await renderAndConnect();

        await act(async () => serverCloseConnection?.(1006));
        await act(async () => vi.advanceTimersByTime(1100));

        await waitFor(() => expect(screen.queryByText(connectingMessage)).not.toBeInTheDocument());
        expect(screen.queryByText(errorMessage)).not.toBeInTheDocument();
    });

    it("retrying after error resets to connecting state", async () => {
        await renderAndConnect();

        server.use(
            tailWs.addEventListener("connection", ({ client }) => {
                client.close(1006);
            }),
        );

        await act(async () => serverCloseConnection?.(1006));
        for (let i = 0; i < 5; i++) {
            await act(async () => vi.advanceTimersByTime(1000 * 2 ** i + 100));
        }

        await waitFor(() => expect(screen.getByText(errorMessage)).toBeInTheDocument());

        server.resetHandlers();
        await act(async () => {
            await userEvent.click(screen.getByRole("button", { name: /retry/i }));
            expect(screen.getByText(connectingMessage)).toBeInTheDocument();
        });
    });
});

describe("LiveTailLogs — receiving logs", () => {
    it("renders an incoming log row", async () => {
        await renderAndConnect();

        await emitLogs({ body: "User logged in" });

        expect(screen.getByText("User logged in")).toBeInTheDocument();
    });

    it("renders multiple logs sorted by timestamp descending", async () => {
        await renderAndConnect();

        await act(async () =>
            sendToClient?.(
                JSON.stringify([
                    {
                        serviceName: "auth-service",
                        records: [
                            { body: "Older log", severityText: "INFO", timestamp: "2024-01-01T10:00:00Z" },
                            { body: "Newer log", severityText: "INFO", timestamp: "2024-01-01T11:00:00Z" },
                        ],
                    },
                ]),
            ),
        );

        const rows = screen.getAllByText(/log/i);
        expect(rows[0].textContent).toContain("Newer log");
        expect(rows[1].textContent).toContain("Older log");
    });
});

describe("LiveTailLogs — pause / resume", () => {
    it("shows pause alert bar when paused", async () => {
        await renderAndConnect();

        await userEvent.click(screen.getByRole("button", { name: /pause/i }));

        expect(screen.getByText(pauseMessage)).toBeInTheDocument();
    });

    it("buffers logs while paused and flushes them on resume", async () => {
        await renderAndConnect();

        await userEvent.click(screen.getByRole("button", { name: /pause/i }));

        await emitLogs({ body: "Buffered log" });
        expect(screen.queryByText("Buffered log")).not.toBeInTheDocument();

        await userEvent.click(screen.getByRole("button", { name: /continue/i }));
        expect(screen.getByText("Buffered log")).toBeInTheDocument();
    });

    it("hides pause bar on resume", async () => {
        await renderAndConnect();

        await userEvent.click(screen.getByRole("button", { name: /pause/i }));
        await userEvent.click(screen.getByRole("button", { name: /continue/i }));

        expect(screen.queryByText(pauseMessage)).not.toBeInTheDocument();
    });

    it("disables pause button while connecting", async () => {
        server.resetHandlers();
        render(<LiveTailLogs />);
        expect(screen.getByRole("button", { name: /pause/i })).toBeDisabled();
    });
});

describe("LiveTailLogs — filters", () => {
    beforeEach(async () => {
        await renderAndConnect();

        await act(async () =>
            sendToClient?.(
                JSON.stringify([
                    {
                        serviceName: "auth-service",
                        records: [{ body: "Auth info log", severityText: "INFO", timestamp: new Date().toISOString() }],
                    },
                    {
                        serviceName: "payment-service",
                        records: [
                            { body: "Payment error log", severityText: "ERROR", timestamp: new Date().toISOString() },
                        ],
                    },
                ]),
            ),
        );

        await waitFor(() => expect(screen.getByText("Auth info log")).toBeInTheDocument());
    });

    it("filters by service", async () => {
        await userEvent.click(screen.getByRole("combobox", { name: /services/i }));
        await userEvent.click(screen.getByRole("option", { name: "auth-service" }));

        expect(screen.getByText("Auth info log")).toBeInTheDocument();
        expect(screen.queryByText("Payment error log")).not.toBeInTheDocument();
    });

    it("filters by severity", async () => {
        await userEvent.click(screen.getByRole("combobox", { name: /severity level/i }));
        await userEvent.click(screen.getByRole("option", { name: "ERROR" }));

        expect(screen.getByText("Payment error log")).toBeInTheDocument();
        expect(screen.queryByText("Auth info log")).not.toBeInTheDocument();
    });

    it("filters by search body with debounce", async () => {
        const searchInput = screen.getByPlaceholderText("Search body...");
        await userEvent.type(searchInput, "Auth");

        expect(screen.getByText("Auth info log")).toBeInTheDocument();

        act(() => vi.advanceTimersByTime(300));

        expect(screen.getByText("Auth info log")).toBeInTheDocument();
        expect(screen.queryByText("Payment error log")).not.toBeInTheDocument();
    });

    it("clears search and shows all logs again", async () => {
        const searchInput = screen.getByPlaceholderText("Search body...");
        await userEvent.type(searchInput, "Auth");
        act(() => vi.advanceTimersByTime(300));

        await userEvent.click(screen.getByTestId("ClearIcon").closest("button")!);
        act(() => vi.advanceTimersByTime(300));

        expect(searchInput).toHaveValue("");
        expect(screen.getByText("Payment error log")).toBeInTheDocument();
    });

    it("shows empty state when no logs match the active filters", async () => {
        await userEvent.click(screen.getByRole("combobox", { name: /severity level/i }));
        await userEvent.click(screen.getByRole("option", { name: "DEBUG" }));

        expect(screen.getByText("No logs match your filters")).toBeInTheDocument();
    });
});
