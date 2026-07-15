import { render, screen, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi, describe, it, expect, beforeEach, afterEach } from "vitest";
import LiveTailLogs from "../../components/LiveTailLogs";
import type { FlatLogRecord } from "../../types/Log";

const connectingMessage = "Connecting to live tail server...";
const pauseMessage = "Live Tail paused — No new logs";
const errorMessage = /live tail server/i;

vi.mock("../../hooks/useDistinctServices", () => ({
    useDistinctServices: () => ({
        data: { services: ["auth-service", "payment-service"] },
        isLoading: false,
    }),
}));

// MOCK WEB WORKER
let mockWorkerInstance: MockWorker | null = null;

class MockWorker {
    onmessage: ((event: MessageEvent) => void) | null = null;
    postMessage = vi.fn();
    terminate = vi.fn();

    constructor() {
        mockWorkerInstance = this;
    }

    // Custom helper to simulate the worker sending a message to the React component
    emit(data: any) {
        if (this.onmessage) {
            act(() => {
                this.onmessage!({ data } as MessageEvent);
            });
        }
    }
}

vi.stubGlobal("Worker", MockWorker);

const makeRecord = (overrides: Partial<FlatLogRecord> = {}): FlatLogRecord => ({
    serviceName: "auth-service",
    body: "User logged in",
    severityText: "INFO",
    timestamp: Date.now(),
    observedTimestamp: Date.now(),
    insertedAt: "2024-01-01T00:00:00Z",
    traceId: "5b8aa5a2d2c8646c14e138a83416a41f",
    spanId: "f96ea2a71a065463",
    severityNumber: 9,
    bodyType: "string",
    scopeName: "",
    scopeVersion: "",
    logAttrKeys: [],
    logAttrValues: [],
    resAttrKeys: [],
    resAttrValues: [],
    ...overrides,
});

/** Renders the component and simulates the worker successfully connecting */
const renderAndConnect = async () => {
    render(<LiveTailLogs />);
    // Simulate the worker telling the UI it connected successfully
    mockWorkerInstance?.emit({ type: "STATUS", payload: "live" });
};

beforeEach(() => {
    mockWorkerInstance = null;
    vi.useFakeTimers({ shouldAdvanceTime: true });
});

afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
});

describe("LiveTailLogs — connection status", () => {
    it("shows connecting bar on initial render", async () => {
        render(<LiveTailLogs />);
        expect(screen.getByText(connectingMessage)).toBeInTheDocument();
    });

    it("hides connecting bar once worker sends connected status", async () => {
        await renderAndConnect();
        expect(screen.queryByText(connectingMessage)).not.toBeInTheDocument();
    });

    it("shows error bar when worker sends error status", async () => {
        await renderAndConnect();
        mockWorkerInstance?.emit({ type: "STATUS", payload: "error" });
        expect(screen.getByText(errorMessage)).toBeInTheDocument();
    });

    it("retrying after error sends RECONNECT command to worker", async () => {
        await renderAndConnect();
        mockWorkerInstance?.emit({ type: "STATUS", payload: "error" });

        await userEvent.click(screen.getByRole("button", { name: /retry/i }));

        // UI should revert to connecting state
        expect(screen.getByText(connectingMessage)).toBeInTheDocument();
        // UI should instruct the worker to reconnect
        expect(mockWorkerInstance?.postMessage).toHaveBeenCalledWith({ type: "RECONNECT" });
    });

    it("sends CLEANUP command to worker on unmount", () => {
        const { unmount } = render(<LiveTailLogs />);
        unmount();

        expect(mockWorkerInstance?.postMessage).toHaveBeenCalledWith({ type: "CLEANUP" });
        expect(mockWorkerInstance?.terminate).toHaveBeenCalled();
    });
});

describe("LiveTailLogs — receiving logs", () => {
    it("renders an incoming log row sent from the worker", async () => {
        await renderAndConnect();

        mockWorkerInstance?.emit({
            type: "LOG_UPDATE",
            payload: [makeRecord({ body: "User logged in" })],
        });

        expect(screen.getByText("User logged in")).toBeInTheDocument();
    });

    it("renders multiple logs provided by the worker", async () => {
        await renderAndConnect();

        mockWorkerInstance?.emit({
            type: "LOG_UPDATE",
            payload: [makeRecord({ body: "Newer test log" }), makeRecord({ body: "Older test log" })],
        });

        const rows = screen.getAllByText(/test log/i);
        expect(rows[0].textContent).toContain("Newer test log");
        expect(rows[1].textContent).toContain("Older test log");
    });
});

describe("LiveTailLogs — pause / resume commands", () => {
    it("shows pause alert bar and sends PAUSE command to worker", async () => {
        const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
        await renderAndConnect();

        await user.click(screen.getByRole("button", { name: /pause/i }));

        expect(screen.getByText(pauseMessage)).toBeInTheDocument();
        expect(mockWorkerInstance?.postMessage).toHaveBeenCalledWith({ type: "PAUSE" });
    });

    it("hides pause bar and sends RESUME command to worker", async () => {
        await renderAndConnect();

        await userEvent.click(screen.getByRole("button", { name: /pause/i }));
        await userEvent.click(screen.getByRole("button", { name: /continue/i }));

        expect(screen.queryByText(pauseMessage)).not.toBeInTheDocument();
        expect(mockWorkerInstance?.postMessage).toHaveBeenCalledWith({ type: "RESUME" });
    });

    it("disables pause button while connecting", async () => {
        render(<LiveTailLogs />);
        expect(screen.getByRole("button", { name: /pause/i })).toBeDisabled();
    });
});

describe("LiveTailLogs — UI filters", () => {
    beforeEach(async () => {
        await renderAndConnect();

        mockWorkerInstance?.emit({
            type: "LOG_UPDATE",
            payload: [
                makeRecord({ serviceName: "auth-service", body: "Auth info log", severityText: "INFO" }),
                makeRecord({ serviceName: "payment-service", body: "Payment error log", severityText: "ERROR" }),
            ],
        });
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

        act(() => vi.advanceTimersByTime(300));

        expect(screen.getByText((_, element) => element?.textContent === "Auth info log")).toBeInTheDocument();
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
